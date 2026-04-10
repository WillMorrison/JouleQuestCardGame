import argparse
import os
from typing import Callable

import numpy as np

from tianshou.data import Collector, VectorReplayBuffer
from tianshou.env import DummyVectorEnv, PettingZooEnv
from tianshou.policy import MultiAgentPolicyManager, DQNPolicy
from tianshou.trainer import OffpolicyTrainer
from tianshou.utils.net.common import Net
from tianshou.utils import TensorboardLogger, LazyLogger

import torch
from torch.utils.tensorboard import SummaryWriter


from game_client import ServerClient
import joulequest_env


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
    )
    policy = MultiAgentPolicyManager([shared_policy] * len(env.agents), env)

    # 3. DATA COLLECTION
    train_envs = DummyVectorEnv([get_env for _ in range(4)])
    test_envs = DummyVectorEnv([get_env])
    
    seed = 1
    np.random.seed(seed)
    torch.manual_seed(seed)
    train_envs.seed(seed)
    test_envs.seed(seed)

    train_collector = Collector(policy, train_envs, VectorReplayBuffer(20000, len(train_envs)), exploration_noise=True)
    test_collector = Collector(policy, test_envs, exploration_noise=True)

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
        max_epoch=2,
        step_per_epoch=128,
        step_per_collect=16,
        episode_per_test=100,
        batch_size=64,
        update_per_step=0.1,
        train_fn=lambda epoch, env_step: shared_policy.set_eps(0.1),
        test_fn=lambda epoch, env_step: shared_policy.set_eps(0.05),
        stop_fn=lambda mean_rewards: mean_rewards >= 10,
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