# deploy

The deployment artefacts an operator actually uses: the container definition, the
composition that stands beside other self-hosted services, and whatever else the
one command needs to exist. Nothing here is compiled into the service and nothing
under `internal/` or `cmd/` may depend on it, so a change to how this is packaged
cannot change how it behaves. It holds no secret, no certificate and no
operator's values, because this directory is public and a deployment artefact
carrying a real deployment's inputs is how those leave a host. What it may assume
about the host it runs on is a closed list in
[the deployment contract](../docs/decisions/deployment-contract.md), and an
artefact here that needs something not on that list is changing that record.
