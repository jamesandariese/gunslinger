# Gunslinger

Shoot mail into gmail directly from a forward in [Mailgun][mailgun].

## Why?

My initial application was [`gunsafe`][gunsafe]([github][github-gs]).  This works
well but I wanted another option to get mail into Gmail rather than self-host the
entire thing.  

Specifically, I wanted to reduce my attack surface by turning off my VPS and hosting
my email service [Google App Engine](https://cloud.google.com/appengine/).  This has
the added benefit of removing a recurring $5/mo charge -- now I can have one extra
Mocha every third month!

The way `gunsafe` worked, it would watch for a notification from Mailgun, download the
message, and save it in Maildir.  Since I've had my email address for nearly 15 years
now, I get a lot of spam so storage on a VPS is annoyingly constrained by this.  I
also don't like deleting things.  Google has a few extra hard drives for me.

## How?

Rather than use a disk as a proxy like `gunsafe`, `gunslinger` uses the
[Gmail Import API][giapi] to receive the MIME body of a message from Mailgun
and send it directly into your inbox.  That's about it!

Go to / and it will auth you to Google and store a refresh token.  Copy the
webhook you're redirected to and put it into Mailgun as a forward.

```
forward("https://your-application-id.appspot.com/webhook/abcd1324#mime")
```

Don't forget the `#mime`!

Now any mail coming into Mailgun will go to your appspot app as a form-encoded
POST body with a mime-body attribute containing the message in RFC 2822 format.

This will trigger the app to send it to the account associated with the oauth
refresh token.

There's no explicit limit to the number of accounts you can use a single appspot
hosted app with but I expect that due to the open source nature of this, it will
likely be one or two (maybe a family?).

## Setup

In the [GCP admin site][gcpconsole], create a new project.
Remember the application ID and replace all the references to `your-application-id`
with it.


```
goapp deploy -application your-application-id -version dev app.yaml
```

In GCP, enable the dev version of your app.

Now browse to https://your-application-id.appspot.com/ and let it auth you to 
Gmail.  You will be redirected to https://your-application-id.appspot.com/webhooks/abcd1234.
Copy the URL from your browser.  You will not have access to it again but you can always
make another.  Still though, keep track of it.

In the mailgun admin page, go to Routes and, at the bottom, use the "Send A Sample POST"
form to test your new URL.  Be sure you add `#mime` to the end of the URL.  This is very
important.

You should receive a message from Mailgun.  If you have a hard time finding it, try
searching Gmail for "mailgun sample post".  Mine was from 4/26/2013 so keep looking.

Assuming all is well, add a `catch_all()` route to your Routes page (if you're not familiar
with this process, it should be the highest number route you have.  Note that if you already
have a catch_all and want to add this as well, follow all these directions but use
`match_recipient(".")` instead of `catch_all()`.

```
Priority:          10  # Or whatever is higher than all the others in your list
Filter Expression: catch_all()
Actions:           forward("https://your-application-id.appspot.com/webhooks/abcd1234#mime")
Description:       A reminder of what this is.  [gunslinger][gunslinger] might be a good idea.
```

Hit save and send a test message!

[gae]: https://cloud.google.com/appengine/
[github-gs]: https://github.com/jamesandariese/gunsafe
[gunslinger]: https://github.com/jamesandariese/gunslinger
[gunsafe]: http://strudelline.net/gunsafe
[mailgun]: https://mailgun.com/
