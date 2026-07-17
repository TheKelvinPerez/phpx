# Elefante Brand Infrastructure

Last updated: July 17, 2026

This is the restart point for reserving and protecting the Elefante brand before product development continues.

## Canonical Brand

Public name: `Elefante`

Infrastructure namespace: `elefantephp`

Domain: `elefantephp.com`

Short description: `The local development runtime for PHP.`

The brand should be Elefante everywhere. Use `elefantephp` wherever a service requires a unique username, organization name, package scope, repository namespace, or account handle.

Product terminology should remain in English:

* Projects
* Workspaces
* Environments
* Services
* Share
* Tunnels

## Already Secured

* [x] Purchased `elefantephp.com` through Porkbun
* [x] Enabled domain auto renewal
* [x] Enabled the domain security lock
* [x] Enabled WHOIS privacy
* [x] Enabled Porkbun API access
* [x] Created the GitHub organization `elefantephp`
* [x] Created the npm organization `elefantephp`
* [x] Reserved the npm package scope `@elefantephp`
* [x] Created `accounts@elefantephp.com`
* [x] Forwarded `accounts@elefantephp.com` to `thekelvinperez@gmail.com`

## Tomorrow's Priority Checklist

Complete these in order. The goal is to claim the important names first. Profiles, artwork, and public announcements can come later.

### 1. Create the Dedicated Elefante Inbox

* [ ] Try to create `elefantephp@proton.me`
* [ ] Use a unique password stored in a password manager
* [ ] Enable two factor authentication or a passkey
* [ ] Save the recovery codes somewhere outside the mailbox
* [ ] Add a trusted recovery method
* [ ] Change the Porkbun forward so `accounts@elefantephp.com` delivers to the new Proton inbox
* [ ] Send a test message from an unrelated email address
* [ ] Confirm the message arrives in Proton

Continue using `accounts@elefantephp.com` when registering services. The Proton address is the private inbox behind it. This keeps every service attached to a domain Elefante owns, instead of attaching the company permanently to one email provider.

Until the Proton inbox is ready and tested, keep the current Gmail forward active.

### 2. Claim the Google Identity

* [ ] Try to claim `elefantephp@gmail.com`
* [ ] Set the Google Account name to `Elefante`
* [ ] Add `accounts@elefantephp.com` as the recovery email
* [ ] Add a trusted recovery phone number
* [ ] Enable two factor authentication or a passkey
* [ ] Save the recovery codes

The Gmail address is worth claiming defensively because it is an obvious Elefante identity. It can remain dedicated to Google services.

YouTube technically requires a Google Account, not a Gmail mailbox. If `elefantephp@gmail.com` is unavailable, create a Google Account using `accounts@elefantephp.com`.

Google Account signup:

https://accounts.google.com/signup

### 3. Create the YouTube Channel

* [ ] Sign in with the dedicated Elefante Google Account
* [ ] Create a business channel or Brand Account
* [ ] Set the channel name to `Elefante`
* [ ] Claim the handle `@elefantephp`
* [ ] Set the website to `https://elefantephp.com`
* [ ] Use the short description `The local development runtime for PHP.`
* [ ] Add the personal Google Account as a second owner or manager
* [ ] Confirm the backup account can access YouTube Studio
* [ ] Do not share the primary account password with collaborators

YouTube channel switcher:

https://www.youtube.com/channel_switcher

### 4. Claim X

* [ ] Create a separate Elefante account
* [ ] Register it with `accounts@elefantephp.com`
* [ ] Claim the username `elefantephp`
* [ ] Set the display name to `Elefante`
* [ ] Set the website to `https://elefantephp.com`
* [ ] Use the short description `The local development runtime for PHP.`
* [ ] Enable two factor authentication
* [ ] Save the recovery codes

Do not rename the personal X account. Elefante should have its own account and ownership history.

X signup:

https://x.com/i/flow/signup

### 5. Claim Docker Hub

* [ ] Create a free individual Docker account
* [ ] Register it with `accounts@elefantephp.com`
* [ ] Claim the Docker ID `elefantephp`
* [ ] Set the display name to `Elefante`
* [ ] Set the website to `https://elefantephp.com`
* [ ] Enable two factor authentication
* [ ] Save the recovery codes

The Docker ID is the important reservation because it becomes part of public image names.

Expected image name:

`docker.io/elefantephp/elefante`

Docker signup:

https://app.docker.com/signup

## Verify Existing Reservations

### GitHub

* [ ] Confirm the organization opens at `https://github.com/elefantephp`
* [ ] Confirm the personal GitHub account is an organization owner
* [ ] Enable required two factor authentication for organization members
* [ ] Add a second trusted owner when one is available
* [ ] Review organization recovery settings
* [ ] Set the display name to `Elefante`
* [ ] Set the website to `https://elefantephp.com`
* [ ] Add the short description

Do not rename or transfer the temporary GitHub repository until the repository migration is handled intentionally.

The eventual canonical repository should be:

`github.com/elefantephp/elefante`

### npm

* [ ] Confirm the organization opens under the `elefantephp` namespace
* [ ] Confirm `thekelvinperez` is an organization owner
* [ ] Enable two factor authentication
* [ ] Review organization recovery settings
* [ ] Do not publish a meaningless placeholder package

Expected future package names:

`@elefantephp/cli`

`@elefantephp/config`

Composer remains responsible for PHP dependency resolution. The npm scope is available for JavaScript packages, editor integrations, frontend tooling, or supporting utilities.

## Additional Brand Surfaces

These come after Proton, Google, YouTube, X, and Docker.

### LinkedIn

* [ ] Create an Elefante company page
* [ ] Request the public identifier `elefantephp`
* [ ] Add `https://elefantephp.com`
* [ ] Use the Elefante name and short description

### Bluesky

* [ ] Reserve `elefantephp.bsky.social` if it is available
* [ ] Later verify `elefantephp.com` and use `@elefantephp.com` as the canonical handle

### Community

* [ ] Enable GitHub Discussions when the repository is ready for public feedback
* [ ] Reserve a Discord community name only when there is a real community to serve
* [ ] Create `security@elefantephp.com` before accepting vulnerability reports
* [ ] Create `support@elefantephp.com` before offering user support

## Distribution Infrastructure

Some namespaces are inherited from the GitHub organization and do not require separate account registration.

### GitHub Container Registry

Canonical image:

`ghcr.io/elefantephp/elefante`

Use GitHub Container Registry as the canonical registry. Docker Hub can mirror the image for discoverability.

### Homebrew

Future tap repository:

`github.com/elefantephp/homebrew-tap`

Expected installation command:

`brew install elefantephp/tap/elefante`

### Go

Future Go module:

`github.com/elefantephp/elefante`

The Go module path follows the final GitHub repository location.

### Composer and Packagist

Future Composer vendor:

`elefantephp/*`

Packagist does not reserve an empty vendor namespace. The namespace becomes associated with Elefante when the first legitimate Composer package is published. Do not publish a fake package solely to reserve the name.

## Long Term Email Setup

The current Porkbun forward is enough for reserving accounts. Later, convert `elefantephp.com` into a proper hosted mailbox through Proton Mail or another provider.

The clean address set is:

* `accounts@elefantephp.com` for ownership, login, and recovery
* `hello@elefantephp.com` for general contact
* `security@elefantephp.com` for security reports
* `support@elefantephp.com` for product support
* `billing@elefantephp.com` for invoices and subscriptions

When moving the custom domain to Proton:

1. Purchase a Proton plan that supports custom domains.
2. Add and verify `elefantephp.com` inside Proton.
3. Create the required addresses before changing DNS.
4. Replace the Porkbun forwarding MX records with Proton's MX records.
5. Configure SPF.
6. Configure DKIM.
7. Configure DMARC.
8. Test incoming mail.
9. Test outgoing mail.
10. Confirm account recovery messages still arrive.

Do not remove the Porkbun forwarding records until the Proton addresses exist and Proton provides the exact replacement DNS records.

## Ownership Standard

Every Elefante account should follow the same rules:

1. Use `accounts@elefantephp.com` as the registration email whenever possible.
2. Use a unique password stored in a password manager.
3. Enable two factor authentication or a passkey.
4. Save recovery codes outside the account being protected.
5. Keep at least one trusted backup owner on services that support multiple owners.
6. Use role based access instead of sharing passwords.
7. Use the personal email and phone only as private recovery methods.
8. Record who owns each account and where its recovery codes are stored.
9. Never make a personal profile the only owner of a critical company asset.

## Canonical Namespace Registry

Brand name: `Elefante`

Universal handle: `elefantephp`

Website: `elefantephp.com`

Ownership email: `accounts@elefantephp.com`

GitHub: `github.com/elefantephp`

npm: `@elefantephp`

X: `@elefantephp`

YouTube: `@elefantephp`

Docker Hub: `elefantephp`

GitHub Container Registry: `ghcr.io/elefantephp`

Composer: `elefantephp/*`

Homebrew tap: `elefantephp/homebrew-tap`

Go module: `github.com/elefantephp/elefante`

## Tomorrow's Minimum Finish Line

If energy is limited, complete only these five things:

1. Create the dedicated Proton inbox.
2. Redirect and test `accounts@elefantephp.com`.
3. Claim the Google identity and YouTube handle.
4. Claim the X handle.
5. Claim the Docker ID.

Once those are secured, the essential Elefante brand infrastructure is protected and product development can continue without rushing the remaining profiles.
