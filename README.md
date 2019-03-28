# go-nntpit

go-nntpit is the primary NNTPit implementation

## what is nntpit?

[Use NNTPit Now](https://nntpit.ronsor.eu.org)

NNTPit is a federated system for sharing posts in similar fashion to sites such as reddit.
Posts, comments, and groups are shared between peers (which you may specify in the configuration
using the `uplinks` option). Votes, however, are not currently shared.

## why would I use it?

The NNTPit network is not run by any single organization and thus one group cannot censor
an idea or message.

## features

* Syncing between peers is done every minute
* Blacklists are available to block unwanted groups and users from being synced.
* Single binary for easy deployment

## dependencies

Go 1.9+
