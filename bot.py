import os

import eth_abi
import requests
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from discord import Intents
from discord.ext import commands
from web3 import Web3

OPT_RPC_ENDPOINT = os.getenv("OPT_RPC_ENDPOINT")
CHANNEL_ID = int(os.getenv("CHANNEL_ID"))
BOT_TOKEN = os.getenv("BOT_TOKEN")

# Velodrome CL1-USDC/sDAI pool on Optimism
POOL = "0x131525f3FA23d65DC2B1EB8B6483a28c43B06916"
# USDC on Optimism
USDC = "0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85"
USDC_DECIMALS = 6
# sDAI on Optimism
SDAI = "0x2218a117083f5B482B0bB821d27056Ba9c04b1D3"
SDAI_DECIMALS = 18

w3 = Web3(Web3.HTTPProvider(OPT_RPC_ENDPOINT))

intents = Intents.default()
intents.messages = True
intents.guilds = True

bot = commands.Bot(command_prefix='!', intents=intents)

@bot.event
async def on_ready():
    print(f'Logged in as {bot.user}!')
    scheduler.start()

def _get_pool_balance(token_address, decimals):
    headers = {
        'Content-Type': 'application/json'
    }
    function_signature = "balanceOf(address)"
    function_selector = w3.keccak(text=function_signature)[:4]
    encoded_arguments = eth_abi.encode(['address'], [POOL])
    data = function_selector.hex() + encoded_arguments.hex()

    payload = {
        "jsonrpc": "2.0",
        "method": "eth_call",
        "params": [
            {
                "to": token_address,
                "data": data
            },
            "latest"
        ],
        "id": 1
    }

    response = requests.post(OPT_RPC_ENDPOINT, headers=headers, json=payload)
    data = response.json()
    result = data["result"]
    balance = int(result, 16) / 10 ** decimals
    return balance

def fetch_and_process_data():
    sdai_balance = _get_pool_balance(SDAI, SDAI_DECIMALS)
    usdc_balance = _get_pool_balance(USDC, USDC_DECIMALS)

    formatted_sdai_balance = f"{int(sdai_balance):,}"
    formatted_usdc_balance = f"{int(usdc_balance):,}"

    message = f"**Velodrome CL1-USDC/sDAI pool**\n- {formatted_sdai_balance} sDAI\n- {formatted_usdc_balance} USDC"

    return message

async def send_data_to_channel():
    channel = bot.get_channel(CHANNEL_ID)
    if channel:
        data = fetch_and_process_data()
        await channel.send(data)
    else:
        print(f"Failed to fetch channel with ID {CHANNEL_ID}")

scheduler = AsyncIOScheduler()
scheduler.add_job(send_data_to_channel, 'interval', hours=6)

bot.run(BOT_TOKEN)
