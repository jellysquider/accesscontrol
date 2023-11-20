# DU Access Control

This repo implements a service that can run on a Raspberry Pi (or similar device) and allows unlocking a door.

The [door](door) package is responsible for controlling the relay (GPIO 21 – maybe this should be configurable?) and unlocking the door.

The [router](router) package serves an API used to trigger unlocking of the door.

## API Overview

For security, all API endpoints are only accessible to users who connect from an IP address matching the environment variable `LOCAL_INTERNET_ADDRESS`. This should be configured as the internet address of the space so that only users connected to the Double Union Wi-Fi and in proximity of the space can unlock the door.

This address could be rotated by our ISP, and it would be ideal for us to dynamically identify this address in the future. The space IP should always available via Dynamic DNS at `doubleunion.tplinkdns.com`

Ideally, the service wouldn't be exposed to the internet at all. Unfortunately (for our niche purposes), Chrome disallows a page served over TLS to make requests to an unencrypted web server. To serve TLS to browsers, we need to used a CA signed certificate. To do that in a free way, certbot is the defacto solution. While it's not impossible to get a certificate from certbot for an intranet server, most automated means of renewing certbot certificates require your server to be exposed to the internet. I've chosen to go with that path for now vs the headache of having to manually renew certificates 4x per year (or have people unexpectedly unable to access the space).

### GET /api/v1/status

This endpoint only enforces the `LOCAL_INTERNET_ADDRESS` check and can be checked to see "am I on the right network to unlock the door". It's useful for a webapp that wants to disable a button and tell you to switch networks.

### POST /api/v1/unlock

Takes a JSON payload in the format:
```json
{
    // a number of seconds between 1-30 before the door should re-lock
    // there is intentionally no way to unlock the door permanently from the api
    "seconds": 10,
}
```

This endpoint additionally requires a signed JWT token that carries a valid subject (`sub`). This is used to validate that only key members – as identifier by [AROOO](https://github.com/doubleunion/arooo) – are authorized to unlock the door. Ostensibly, anyone with access to the signing key on the Double Union network would also have this ability, so it should be kept secure.

The signing key is configured by the environment variable `ACCESS_CONTROL_SIGNING_KEY`. AROOO is configured with the same signing key and [provides an API](https://github.com/doubleunion/arooo/blob/main/app/controllers/members/access_controller.rb) for key members to generate short-lived tokens for unlocking the door.

When a given subject locks or unlocks the door it is logged for auditing purposes.

# Notes on Space Configuration

On the Pi in the space, this service is run by systemd and called `accesscontrol.service`.

## Managing the service

* **Start:** `sudo systemctl start accesscontrol.service`
* **Stop:** `sudo systemctl stop accesscontrol.service`
* **Restart:** `sudo systemctl restart accesscontrol.service`

## Viewing Logs

* **Tail:** `journalctl -f -u accesscontrol.service`
