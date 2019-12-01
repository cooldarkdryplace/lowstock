package lowstock

var (
	helpMsg = `Supported commands:

/start	- Login to your Etsy shop
/pin	- Submit login Pin
/help	- Send this message`

	startMsg = `<b>Welcome to the Lowstock!</b>

This bot keeps track of your Etsy listings and informs you when the listing is sold-out.
Before you start getting notifications, you need to log in.
This application will request read-only access to your shop information and your listings.
This app stores a minimal amount of data needed for notification functionality: your Etsy user id and access token.

After you have logged in into your Etsy account and authorized this app - you will get a one-time pin code.

Please submit this code to this chat in a form:
<code>/pin {pin code}</code>

Example:
<code>/pin 76279961</code>

Type /help to get the list of commands, or check the <a href="">online documentation</a>.

This bot is opensource. You can find <a href="https://github.com/cooldarkdryplace/lowstock">source code</a> on Github.
<i>The term 'Etsy' is a trademark of Etsy, Inc. This application uses the Etsy API but is not endorsed or certified by Etsy, Inc.</i>`

	successMsg = `Success!
You will be notified when products are sold out.`
)
