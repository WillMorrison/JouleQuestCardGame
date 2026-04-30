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


@dataclasses.dataclass(frozen=True, kw_only=True)
class HyperParameters:
    num_players: int = 4
    num_envs: int = 2

    # Trainer parameters
    max_epochs: int = 1250
    step_per_epoch: int = 10000
    step_per_collect: int = 400
    episode_per_test: int = 100
    batch_size: int = 1000
    update_per_step: float = 0.5
    # This controls the exploration rate of the policy. Higher epsilon means more random actions.
    epsilon_start: float = 0.2
    epsilon_end: float = 0.001
    epsilon_decay_exp: float = 0.2
    epsilon_test: float = 0.001
    stop_reward: float = 50 # Stop training once we reach this average reward over the test episodes

    # Replay buffer parameters
    buffer_length: int = 160000
    # Beta controls the importance sampling weights for the prioritized replay buffer.
    # Higher beta means more correction for the bias introduced by prioritization.
    beta_start: float = 0.4
    beta_end: float = 1.0

    # Policy parameters
    discount_factor: float = 0.995 # Higher discount factor for longer-term rewards
    estimation_step: int = 20 # How many steps to look ahead when calculating the target Q value. Higher means better long-term planning but more variance.

def train(get_env: Callable[[], PettingZooEnv], logger: BaseLogger, warmup_config: WarmupBufferConfig, hparams: HyperParameters) -> BasePolicy:
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
        discount_factor=hparams.discount_factor, # Higher discount factor for longer-term rewards
        estimation_step=hparams.estimation_step, # How many steps to look ahead when calculating the target Q value. Higher means better long-term planning but more variance.
    )
    policy = MultiAgentPolicyManager([shared_policy] * hparams.num_players, env)

    # DATA COLLECTION
    train_envs = DummyVectorEnv([get_env for _ in range(hparams.num_envs)])
    test_envs = DummyVectorEnv([get_env for _ in range(hparams.num_envs)])
    
    seed = 1
    np.random.seed(seed)
    torch.manual_seed(seed)
    train_envs.seed(seed)
    test_envs.seed(seed)

    train_buffer = PrioritizedVectorReplayBuffer(hparams.buffer_length, hparams.num_envs, alpha=0.6, beta=hparams.beta_start, weight_norm=False)
    train_collector = Collector(policy, train_envs, train_buffer, exploration_noise=True)
    test_collector = Collector(policy, test_envs, exploration_noise=True)

    if warmup_config.load and warmup_config.path:
        print("Loading replay buffer from", warmup_config.path)
        train_buffer=train_buffer.load_hdf5(warmup_config.path)
        print(f"Loaded replay buffer with {len(train_buffer)} samples")
    else:
        warmup_config.preheat(train_buffer, train_envs)

    def train_fn(epoch, env_step):
        progress = env_step / (hparams.max_epochs * hparams.step_per_epoch)

        # Epsilon decay from 0.4 to 0.02 over the course of training
        # This controls the exploration rate of the policy. Higher epsilon means more random actions.
        eps = hparams.epsilon_start + (hparams.epsilon_end-hparams.epsilon_start) * progress**hparams.epsilon_decay_exp
        shared_policy.set_eps(eps)

        # Beta increase from 0.4 to 1.0 over the course of training
        # This controls the importance sampling weights for the prioritized replay buffer. Higher beta means more correction for the bias introduced by prioritization.
        beta = hparams.beta_start + (hparams.beta_end-hparams.beta_start) * (progress)
        train_buffer.set_beta(beta)

        logger.write(step_type="train", data={"train/epsilon": eps, "train/beta": beta}, step=env_step)

    # TRAINING LOOP
    OffpolicyTrainer(
        policy=policy,
        train_collector=train_collector,
        test_collector=test_collector,
        max_epoch=hparams.max_epochs,
        step_per_epoch=hparams.step_per_epoch,
        step_per_collect=hparams.step_per_collect,
        episode_per_test=hparams.episode_per_test,
        batch_size=hparams.batch_size,
        update_per_step=hparams.update_per_step,
        train_fn=train_fn,
        test_fn=lambda epoch, env_step: shared_policy.set_eps(hparams.epsilon_test),
        stop_fn=lambda mean_rewards: mean_rewards >= hparams.stop_reward,
        logger = logger,
    ).run()

    return shared_policy


def _get_action(policy: BasePolicy, agent, obs_dict):
    # Run a forward pass
    batch = Batch(obs=[obs_dict["observation"]], info=[{"action_mask": obs_dict["action_mask"]}])
    with torch.no_grad():
        result = policy(batch)
    
    # Apply mask to logits and select the highest valued action.
    mask = obs_dict["action_mask"]
    masked_logits = result.logits[0].numpy()
    masked_logits[np.logical_not(mask)] = -1e10
    action = np.argmax(masked_logits)
    print(f"Agent {agent} obs: {obs_dict}, action: {action}, logits: {masked_logits}")
    return action

def _run_demo(env: joulequest_env.JoulequestEnv, policy: BasePolicy):
    policy.eval()
    env.reset()
    for agent in env.agent_iter(max_iter=250):
        observation, _, termination, truncation, _ = env.last()

        if termination or truncation:
            action = None
        else:
            action = _get_action(policy, agent, observation)

        env.step(action)
    print("Demo complete. Final game state:")
    print(env.get_log())


def main():
    run_id = datetime.datetime.now(datetime.UTC).strftime("%Y-%m-%d_%H-%M-%S")

    parser = argparse.ArgumentParser(
                    prog='JouleQuest policy trainer',
                    description='Train a MARL model to play JouleQuest')
    parser.add_argument('--executable', required=True, help="Path to the rest_api executable")

    # Training
    parser.add_argument('--num_players', default=HyperParameters().num_players, type=int, help='Number of players per game')
    parser.add_argument('--num_envs', default=HyperParameters().num_envs, type=int, help='Number of parallel environments to use for training and testing')
    parser.add_argument('--max_epochs', default=HyperParameters().max_epochs, type=int, help='Number of training epochs to run')
    parser.add_argument('--step_per_epoch', default=HyperParameters().step_per_epoch, type=int, help='Number of environment steps to run per epoch')
    parser.add_argument('--step_per_collect', default=HyperParameters().step_per_collect, type=int, help='Number of environment steps to run for each data collection phase')
    parser.add_argument('--episode_per_test', default=HyperParameters().episode_per_test, type=int, help='Number of episodes to run for each test phase')
    parser.add_argument('--batch_size', default=HyperParameters().batch_size, type=int, help='Batch size for policy updates')
    parser.add_argument('--update_per_step', default=HyperParameters().update_per_step, type=float, help='Number of policy updates to perform per environment step.')
    parser.add_argument('--epsilon_start', default=HyperParameters().epsilon_start, type=float, help='Starting value of epsilon for epsilon-greedy exploration')
    parser.add_argument('--epsilon_end', default=HyperParameters().epsilon_end, type=float, help='Final value of epsilon for epsilon-greedy exploration')
    parser.add_argument('--epsilon_decay_exp', default=HyperParameters().epsilon_decay_exp, type=float, help='Exponent for epsilon decay schedule 1 is linear, 0.5 is sqrt, etc.')
    parser.add_argument('--epsilon_test', default=HyperParameters().epsilon_test, type=float, help='Value of epsilon to use during testing')
    parser.add_argument('--stop_reward', default=HyperParameters().stop_reward, type=float, help='Stop training once the average reward over the test episodes reaches this value')
    parser.add_argument('--discount_factor', default=HyperParameters().discount_factor, type=float, help='Discount factor for future rewards (gamma)')
    parser.add_argument('--estimation_step', default=HyperParameters().estimation_step, type=int, help='Number of steps to look ahead when calculating the target Q value (n-step returns)')
    parser.add_argument('--buffer_length', default=HyperParameters().buffer_length, type=int, help='Maximum number of transitions to store in the replay buffer')
    parser.add_argument('--beta_start', default=HyperParameters().beta_start, type=float, help='Starting value of beta for prioritized replay buffer importance sampling weights')
    parser.add_argument('--beta_end', default=HyperParameters().beta_end, type=float, help='Final value of beta for prioritized replay buffer importance sampling weights')

    # Warmup
    parser.add_argument('--replay_buffer', default="", type=pathlib.Path, help="Path to an HDF5 file to load the replay buffer from, or with --warmup_episodes, the path to save the buffer to for later replaying")
    parser.add_argument('--warmup_episodes', default=0, type=int, help="Number of episodes to pre-fill the replay buffer with before starting training")

    # Logging
    parser.add_argument('--tensorboard_dir', default=f"log/{run_id}/", type=pathlib.Path, help='Path to log statistics to, which can be visualized with tensorboard')

    # Model saving
    parser.add_argument('--save', default=f"agent_{run_id}.pt", type=pathlib.Path, help='Path to save the trained model to')
    parser.add_argument('--run_demo', default=True, action='store_true', help='Whether to run a demo of the trained policy after training completes')

    args = parser.parse_args()

    hparams=HyperParameters(
        num_players=args.num_players,
        num_envs=args.num_envs,
        max_epochs=args.max_epochs,
        step_per_epoch=args.step_per_epoch,
        step_per_collect=args.step_per_collect,
        episode_per_test=args.episode_per_test,
        batch_size=args.batch_size,
        update_per_step=args.update_per_step,
        epsilon_start=args.epsilon_start,
        epsilon_end=args.epsilon_end,
        epsilon_decay_exp=args.epsilon_decay_exp,
        epsilon_test=args.epsilon_test,
        stop_reward=args.stop_reward,
        discount_factor=args.discount_factor,
        estimation_step=args.estimation_step,
        buffer_length=args.buffer_length,
        beta_start=args.beta_start,
        beta_end=args.beta_end,
        )

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
        writer = SummaryWriter(str(args.tensorboard_dir))
        writer.add_text(f"warmup", str(warmup_config))
        writer.add_text(f"hparams", str(hparams))
        logger = TensorboardLogger(writer)
    else:
        logger = LazyLogger()

    # Train the policy
    with ServerClient(args.executable, socket_path="/tmp/joulequest_api_train.sock", suppress_output=True) as cl:
        shared_policy = train(
            get_env=lambda: PettingZooEnv(joulequest_env.env(num_players=args.num_players, client=cl)),
            logger=logger, 
            warmup_config=warmup_config,
            hparams=hparams,
        )
        if args.run_demo:
            print("Running demo of trained policy...")
            demo_env = joulequest_env.JoulequestEnv(num_players=args.num_players, client=cl)
            _run_demo(demo_env, shared_policy)
    
    shared_policy.eval()
    torch.save(shared_policy.state_dict(), args.save)
    print("Training Complete. Model saved to", args.save)

if __name__ == "__main__":
    main()