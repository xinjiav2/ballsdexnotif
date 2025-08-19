
import discord
from discord.ext import commands
import os

# Bot configuration
TRIGGER_PHRASE = "important announcement"  # Change this to your desired phrase
USERS_TO_PING = [123456789012345678, 987654321098765432]  # Replace with actual user IDs

# Bot setup
intents = discord.Intents.default()
intents.message_content = True  # Required to read message content
bot = commands.Bot(command_prefix='!', intents=intents)

@bot.event
async def on_ready():
    print(f'{bot.user} has connected to Discord!')
    print(f'Bot is in {len(bot.guilds)} guilds')

@bot.event
async def on_message(message):
    # Don't respond to the bot's own messages
    if message.author == bot.user:
        return
    
    # Check if the trigger phrase is in the message (case-insensitive)
    if TRIGGER_PHRASE.lower() in message.content.lower():
        # Create mentions for specified users
        mentions = []
        for user_id in USERS_TO_PING:
            user = bot.get_user(user_id)
            if user:
                mentions.append(user.mention)
        
        if mentions:
            # Send the ping message
            ping_message = f"🔔 Attention: {' '.join(mentions)} - The phrase '{TRIGGER_PHRASE}' was mentioned!"
            await message.channel.send(ping_message)
            
            # Optional: React to the original message
            await message.add_reaction('📢')
    
    # Process commands (important for other bot functionality)
    await bot.process_commands(message)

# Optional: Command to add users to ping list
@bot.command(name='add_ping')
@commands.has_permissions(administrator=True)
async def add_ping_user(ctx, user: discord.User):
    """Add a user to the ping list (Admin only)"""
    if user.id not in USERS_TO_PING:
        USERS_TO_PING.append(user.id)
        await ctx.send(f"Added {user.mention} to the ping list!")
    else:
        await ctx.send(f"{user.mention} is already in the ping list!")

# Optional: Command to remove users from ping list
@bot.command(name='remove_ping')
@commands.has_permissions(administrator=True)
async def remove_ping_user(ctx, user: discord.User):
    """Remove a user from the ping list (Admin only)"""
    if user.id in USERS_TO_PING:
        USERS_TO_PING.remove(user.id)
        await ctx.send(f"Removed {user.mention} from the ping list!")
    else:
        await ctx.send(f"{user.mention} is not in the ping list!")

# Optional: Command to list current ping users
@bot.command(name='ping_list')
async def show_ping_list(ctx):
    """Show current users in the ping list"""
    if not USERS_TO_PING:
        await ctx.send("No users in the ping list!")
        return
    
    users = []
    for user_id in USERS_TO_PING:
        user = bot.get_user(user_id)
        if user:
            users.append(user.mention)
    
    if users:
        await ctx.send(f"Current ping list: {', '.join(users)}")
    else:
        await ctx.send("No valid users found in the ping list!")

# Optional: Command to change the trigger phrase
@bot.command(name='set_phrase')
@commands.has_permissions(administrator=True)
async def set_trigger_phrase(ctx, *, phrase):
    """Set the trigger phrase (Admin only)"""
    global TRIGGER_PHRASE
    TRIGGER_PHRASE = phrase
    await ctx.send(f"Trigger phrase changed to: '{phrase}'")

# Run the bot
if __name__ == "__main__":
    # Get token from environment variable for security
    TOKEN = os.getenv('DISCORD_BOT_TOKEN')
    
    if not TOKEN:
        print("Please set the DISCORD_BOT_TOKEN environment variable")
        print("Or replace this with: bot.run('YOUR_BOT_TOKEN_HERE')")
    else:
        bot.run(TOKEN)
