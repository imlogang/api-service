# api-service

This api-service is a service written to be used as a backend to a Postgres deployment for my backend.

It is primarily written with an o11y first guideline utilizing the [ex](https://github.com/circleci/ex) library and deployed via its [helm chart](https://github.com/imlogang/go-api-helm) to my local Kubernetes cluster.
Once there, the main consumer is a Discord bot I wrote with [DiscordJS](https://discord.js.org/).

O11y is observed through Honeycomb allowing tracing through the API endpoints.