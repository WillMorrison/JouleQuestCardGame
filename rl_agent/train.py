import argparse
import os
from typing import Any, Callable

import numpy as np

from tianshou.data import Collector, PrioritizedVectorReplayBuffer, Batch, ReplayBuffer
from tianshou.env import DummyVectorEnv, PettingZooEnv, BaseVectorEnv
from tianshou.policy import MultiAgentPolicyManager, DQNPolicy, BasePolicy
from tianshou.trainer import OffpolicyTrainer
from tianshou.utils.net.common import Net
from tianshou.utils import TensorboardLogger, LazyLogger

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
    
def preheat_buffer(buffer: ReplayBuffer, env: BaseVectorEnv):
    """Preheat by filling the buffer with random actions, respecting the action masks and avoiding stupid actions when possible."""
    warmup_collector = Collector(RandomMaskedPolicy(), env, buffer)
    warmup_collector.collect(n_episode=1000, random=True)

def train(get_env: Callable[[], PettingZooEnv], writer: SummaryWriter|None = None):
    env = get_env()

    # 1. THE BRAIN (Neural Network)
    observation_shape = joulequest_env.OBSERVATION_SPACE["observation"].shape
    action_shape = joulequest_env.ACTION_SPACE.n
    net = Net(
        state_shape=observation_shape,
        action_shape=action_shape,
        hidden_sizes=[128, 128],
        device="cuda" if torch.cuda.is_available() else "cpu",
    ).to("cuda" if torch.cuda.is_available() else "cpu")
    optim = torch.optim.Adam(net.parameters(), lr=1e-4)

    # 2. THE POLICY (DQN)
    # This policy will be shared by all agents (Parameter Sharing)
    shared_policy = DQNPolicy(
        model=net, 
        optim=optim,
        discount_factor=0.995, # Higher discount factor for longer-term rewards
        estimation_step=20, # How many steps to look ahead when calculating the target Q value. Higher means better long-term planning but more variance.
    )
    policy = MultiAgentPolicyManager([shared_policy] * len(env.agents), env)

    # 3. DATA COLLECTION
    train_envs = DummyVectorEnv([get_env for _ in range(2)])
    test_envs = DummyVectorEnv([get_env for _ in range(2)])
    
    seed = 1
    np.random.seed(seed)
    torch.manual_seed(seed)
    train_envs.seed(seed)
    test_envs.seed(seed)

    train_buffer = PrioritizedVectorReplayBuffer(160000, len(train_envs), alpha=0.6, beta=0.4, weight_norm=True)
    train_collector = Collector(policy, train_envs, train_buffer, exploration_noise=True)
    test_collector = Collector(policy, test_envs, exploration_noise=True)

    preheat_buffer(train_buffer, train_envs)

    max_epoch = 250
    def train_fn(epoch, env_step):
        # Epsilon decay from 0.4 to 0.1 over the course of training
        # This controls the exploration rate of the policy. Higher epsilon means more random actions.
        eps = 0.4 - 0.3 * (epoch / max_epoch)
        shared_policy.set_eps(eps)

        # Beta increase from 0.4 to 1.0 over the course of training
        # This controls the importance sampling weights for the prioritized replay buffer. Higher beta means more correction for the bias introduced by prioritization.
        beta = 0.4 + 0.6 * (epoch / max_epoch)
        train_buffer.set_beta(beta)

    # 4. Logging

    if writer is not None:
        logger = TensorboardLogger(writer)
    else:
        logger = LazyLogger()

    # 5. TRAINING LOOP
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
    parser = argparse.ArgumentParser(
                    prog='JouleQuest server runner',
                    description='Runs the joulequest server in a child process and communicates with it over a unix socket')
    parser.add_argument('--executable', required=True, help="Path to the rest_api executable")
    parser.add_argument('--num_players', default=4, type=int, help='Number of players per game')
    parser.add_argument('--tensorboard_dir', default="", type=str, help='Path to log statistics to, which can be visualized with tensorboard')
    parser.add_argument('--save', default='agent.pt', type=str, help='Path to save the trained model to')
    args = parser.parse_args()

    log_writer = None
    if args.tensorboard_dir:
        os.makedirs(args.tensorboard_dir, exist_ok=True)
        log_writer = SummaryWriter(args.tensorboard_dir)

    with ServerClient(args.executable, socket_path="/tmp/joulequest_api_train.sock", suppress_output=True) as cl:
        shared_policy = train(
            get_env=lambda: PettingZooEnv(joulequest_env.env(num_players=args.num_players, client=cl)),
            writer=log_writer,
        )
    
    shared_policy.eval()
    torch.save(shared_policy.state_dict(), args.save)
    print("Training Complete. Model saved to", args.save)

if __name__ == "__main__":
    main()