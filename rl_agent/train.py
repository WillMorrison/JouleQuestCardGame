import argparse
import dataclasses
import datetime
import os
import pathlib
from typing import Any, Callable

import numpy as np

from tianshou.data import Collector, PrioritizedVectorReplayBuffer, Batch, ReplayBuffer
from tianshou.env import DummyVectorEnv, PettingZooEnv, BaseVectorEnv
from tianshou.policy import MultiAgentPolicyManager, DQNPolicy, BasePolicy
from tianshou.trainer import OffpolicyTrainer
from tianshou.utils.net.common import Net
from tianshou.utils import TensorboardLogger, LazyLogger, BaseLogger

import torch
from torch.utils.tensorboard import SummaryWriter


from game_client import ServerClient
import joulequest_env


class RandomMaskedPolicy(BasePolicy):
    def forward(self, batch, state=None, **kwargs):
        # Extract the mask from the observation batch
        mask = batch.obs.mask

        # Filter out actions considered "stupid" by applying a fixed mask, but only if it doesn't completely eliminate all options
        ok_actions = np.array([1, 1, 0, 0, 0, 1, 1, 1, 1, 0, 0, 1, 1, 1, 1], dtype=np.int8)
        filtered = mask*ok_actions
        if not np.array_equal(filtered, np.zeros(filtered.shape)):
            mask = filtered

        # Generate random logits, then set masked actions to a very low number
        logits = torch.randn(mask.shape)
        logits[mask == 0] = -1e10 
        return Batch(act=logits.argmax(dim=-1), state=state)
    
    def learn(self, batch: Batch, **kwargs: Any) -> dict[str, Any]:
        return {}


@dataclasses.dataclass(frozen=True)
class WarmupBufferConfig:
    path: str = ""
    load: bool = False
    save: bool = False
    warmup_episodes: int = 0

    def preheat(self, buffer: ReplayBuffer, env: BaseVectorEnv):
        """Preheat by filling the buffer with random actions, respecting the action masks and avoiding stupid actions when possible."""
        if self.warmup_episodes:
            print(f"Pre-filling replay buffer with {self.warmup_episodes} episodes")
            warmup_collector = Collector(RandomMaskedPolicy(), env, buffer)
            warmup_collector.collect(n_episode=self.warmup_episodes)
            if self.save and self.path:
                print(f"Saving replay buffer with {len(buffer)} samples to {self.path}")
                buffer.save_hdf5(self.path)


def train(get_env: Callable[[], PettingZooEnv], logger: BaseLogger, warmup_config: WarmupBufferConfig, max_epoch: int) -> BasePolicy:
    env = get_env()

    # THE BRAIN (Neural Network)
    observation_shape = joulequest_env.OBSERVATION_SPACE["observation"].shape
    action_shape = joulequest_env.ACTION_SPACE.n
    net = Net(
        state_shape=observation_shape,
        action_shape=action_shape,
        hidden_sizes=[128, 128],
        device="cuda" if torch.cuda.is_available() else "cpu",
    ).to("cuda" if torch.cuda.is_available() else "cpu")
    optim = torch.optim.Adam(net.parameters(), lr=1e-4)

    # THE POLICY (DQN)
    # This policy will be shared by all agents (Parameter Sharing)
    shared_policy = DQNPolicy(
        model=net, 
        optim=optim,
        discount_factor=0.995, # Higher discount factor for longer-term rewards
        estimation_step=20, # How many steps to look ahead when calculating the target Q value. Higher means better long-term planning but more variance.
    )
    policy = MultiAgentPolicyManager([shared_policy] * len(env.agents), env)

    # DATA COLLECTION
    train_envs = DummyVectorEnv([get_env for _ in range(2)])
    test_envs = DummyVectorEnv([get_env for _ in range(2)])
    
    seed = 1
    np.random.seed(seed)
    torch.manual_seed(seed)
    train_envs.seed(seed)
    test_envs.seed(seed)

    train_buffer = PrioritizedVectorReplayBuffer(160000, len(train_envs), alpha=0.6, beta=0.4, weight_norm=False)
    train_collector = Collector(policy, train_envs, train_buffer, exploration_noise=True)
    test_collector = Collector(policy, test_envs, exploration_noise=True)

    if warmup_config.load and warmup_config.path:
        print("Loading replay buffer from", warmup_config.path)
        train_buffer=train_buffer.load_hdf5(warmup_config.path)
        print(f"Loaded replay buffer with {len(train_buffer)} samples")
    else:
        warmup_config.preheat(train_buffer, train_envs)

    def train_fn(epoch, env_step):
        # Epsilon decay from 0.4 to 0.1 over the course of training
        # This controls the exploration rate of the policy. Higher epsilon means more random actions.
        eps = 0.4 - 0.3 * (epoch / max_epoch)
        shared_policy.set_eps(eps)

        # Beta increase from 0.4 to 1.0 over the course of training
        # This controls the importance sampling weights for the prioritized replay buffer. Higher beta means more correction for the bias introduced by prioritization.
        beta = 0.4 + 0.6 * (epoch / max_epoch)
        train_buffer.set_beta(beta)

    # TRAINING LOOP
    OffpolicyTrainer(
        policy=policy,
        train_collector=train_collector,
        test_collector=test_collector,
        max_epoch=max_epoch,
        step_per_epoch=10000,
        step_per_collect=100,
        episode_per_test=100,
        batch_size=200,
        update_per_step=0.1,
        train_fn=train_fn,
        test_fn=lambda epoch, env_step: shared_policy.set_eps(0.01),
        stop_fn=lambda mean_rewards: mean_rewards >= 50,
        logger = logger,
    ).run()

    return shared_policy


def main():
    run_id = datetime.datetime.now(datetime.UTC).strftime("%Y-%m-%d_%H-%M-%S")

    parser = argparse.ArgumentParser(
                    prog='JouleQuest policy trainer',
                    description='Train a MARL model to play JouleQuest')
    parser.add_argument('--executable', required=True, help="Path to the rest_api executable")

    # Training
    parser.add_argument('--num_players', default=4, type=int, help='Number of players per game')
    parser.add_argument('--epochs', default=1250, type=int, help='Number of training epochs to run')

    # Warmup
    parser.add_argument('--replay_buffer', default="", type=pathlib.Path, help="Path to an HDF5 file to load the replay buffer from, or with --warmup_episodes, the path to save the buffer to for later replaying")
    parser.add_argument('--warmup_episodes', default=0, type=int, help="Number of episodes to pre-fill the replay buffer with before starting training")

    # Logging
    parser.add_argument('--tensorboard_dir', default=f"log/{run_id}/", type=pathlib.Path, help='Path to log statistics to, which can be visualized with tensorboard')

    # Model saving
    parser.add_argument('--save', default=f"agent_{run_id}.pt", type=pathlib.Path, help='Path to save the trained model to')

    args = parser.parse_args()

    # Warmup buffer load/preheat/save
    if args.replay_buffer.suffix == '.h5' and args.warmup_episodes > 0:
        warmup_config = WarmupBufferConfig(path=str(args.replay_buffer), save=True, warmup_episodes=args.warmup_episodes)
    elif args.replay_buffer.suffix == '.h5' and args.replay_buffer.exists():
        warmup_config = WarmupBufferConfig(path=str(args.replay_buffer), load=True)
    elif args.warmup_episodes > 0:
        warmup_config = WarmupBufferConfig(warmup_episodes=args.warmup_episodes)
    else:
        warmup_config = WarmupBufferConfig()

    # Logger setup
    if not args.tensorboard_dir.exists() or args.tensorboard_dir.is_dir():
        print(f"Logging to directory {args.tensorboard_dir}")
        args.tensorboard_dir.mkdir(parents=True, exist_ok=True)
        logger = TensorboardLogger(SummaryWriter(str(args.tensorboard_dir)))
    else:
        logger = LazyLogger()

    # Train the policy
    with ServerClient(args.executable, socket_path="/tmp/joulequest_api_train.sock", suppress_output=True) as cl:
        shared_policy = train(
            get_env=lambda: PettingZooEnv(joulequest_env.env(num_players=args.num_players, client=cl)),
            logger=logger, 
            warmup_config=warmup_config,
            max_epoch=args.epochs,
        )
    
    shared_policy.eval()
    torch.save(shared_policy.state_dict(), args.save)
    print("Training Complete. Model saved to", args.save)

if __name__ == "__main__":
    main()