# Electric Borneo Cat - Low stock notifier

Disclaimer:
The term 'Etsy' is a trademark of Etsy, Inc. This application uses the Etsy API but is not endorsed or certified by Etsy, Inc.

## TL;DR

You have an Etsy shop, and you do not want to miss when a listing goes out of stock?
This Telegram bot will notify you when the product is sold out.

  * [Start bot](#start-bot)
  * [How it works](#how-it-works)
     * [Listings Feed](#listings-feed)
     * [Storage](#storage)
     * [Registered users](#registered-users)
     * [Notifications](#notifications)
        * [Telegram](#telegram)
  * [Deployment](#deployment)
     * [APIs access](#apis-access)
  * [Scaling](#scaling)

## Start bot
Start a conversation with [@lowstockbot](https://telegram.me/lowstockbot) on Telegram. You will immediately get a response with the introduction and invitation to login to Etsy.  

![Start Lowstock Bot](https://storage.googleapis.com/lowstock/1_start_bot.jpg)  

Follow the link.  

![Follow the link](https://storage.googleapis.com/lowstock/2_login_url.jpg)  

Etsy will ask you if it is ok to read your listings.  

![Check permissions](https://storage.googleapis.com/lowstock/3_etsy_auth.jpg)  

Once you have approved the requested access, you will get a pin.  

![Etsy pin](https://storage.googleapis.com/lowstock/4_one_time_pin.jpg)  

Copy the pin.  

![Copy pin](https://storage.googleapis.com/lowstock/5_copy_pin.jpg)  

Post the pin back to the chat.  

![Paste pin](https://storage.googleapis.com/lowstock/6_paste_pin.jpg)  

Bot expects this format:  
```
/pin {one time pin}
```  

![Pasted pin](https://storage.googleapis.com/lowstock/7_pasted_pin.jpg)  

The bot will validate your pin, and if everything is fine, you will get a confirmation that from now on, a message will be sent to notify you if listing in your shop is out of stock.  

![Success](https://storage.googleapis.com/lowstock/8_success.jpg)  

For help use `/help` command.  

## How it works
Lowstok listens to all Etsy Listing updates by polling live feeds endpoint.

### Listings Feed
There are two options when it comes to consuming listings updates:
* Push
* Poll

Push makes things simple as you only handle incoming HTTP requests and act accordingly based on the updated type.
I did not manage to find a way to subscribe to feeds reliably, so I have decided to consume primary listings feed by regularly polling endpoint.

### Storage
Lowstock mostly reads data from storage and only stores data when a new user joins.
For simplicity, Lowstock uses local file-based embedded database BoltDB.

### Registered users
Application stores IDs of registered users. Once bot encounters an update that has known user ID - it will send a notification to a corresponding chat.

### Notifications
Notifications are simple text payloads with SKU info and shop name. 

#### Telegram
The current implementation uses Telegram for notifications. It should be easy to plug any other messenger that has API.

## Deployment
You can find a Systemd service unit configuration in this repository.
It is also ok to run this bot in Docker, but you will need to write a Dockerfile yourself.

### APIs access
Lowstock needs access to Etsy Feeds, Etsy Open API, and Telegram Bot API to work.
Once you provision API credentials, you will need to provide them to the bot via environment variables.

| Name                 | Description                                                     |
|----------------------|-----------------------------------------------------------------|
| `DATABASE_FILE`      | BoltDB database file                                            |
| `TELEGRAM_TOKEN`     | Telegram Bot token                                              |
| `ETSY_CONSUMER_KEY`  | Etsy key is used to perform calls to Etsy Open API              |
| `ETSY_SHARED_SECRET` | Etsy secret is used in combination with the key to do OAuth v1  |

## Scaling
With the current number of listing updates per minute, you do not need more than one worker.
You may want to have more for redundancy, this is not done, and I do not think it is is necessary now.
With the current implementation, you need to make sure that it is restarted if failed. System or Docker works for that.
