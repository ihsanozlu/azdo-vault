# AzDO Vault

**AzDO Vault** is a cross-organization Azure DevOps backup & migration CLI.
It lets you **backup**, **restore**, and **migrate** Azure DevOps resources safely between projects and organizations, using local JSON/git-based backups.

> Designed for DevOps engineers who need **repeatable**, **idempotent**, and **auditable** Azure DevOps migrations.

---

## Why AzDO Vault?

Azure DevOps does not provide a native, complete migration mechanism across organizations. AzDO Vault fills that gap by acting as a **migration engine** with a **local state**.

Key goals:

* Cross-organization & cross-project migrations
* Safe re-runs (idempotent where possible)
* Human-readable local backups
* Works with existing `az` authentication
* Scriptable & automation-friendly

---

## Supported Resources

### Backup & Restore

* Git repositories (mirror clone + create + push)
* Branch policies
* Classic build definitions
* Classic release definitions
* YAML pipelines
* Task groups
* Service connections
* Variable groups
* Azure Artifacts feeds
* Azure DevOps Wikis

### Utility Commands

* List artifacts feeds, packages, versions
* Create missing repositories
* Set default branch for repositories

---

## Installation

### From source

```bash
git clone https://github.com/ihsanozlu/azdo-vault.git
cd azdo-vault
go build -o azdo-vault
```

---

## Prerequisites

* Go >= 1.24
* Azure DevOps access
* Azure CLI installed and authenticated:

```bash
az login
az extension add --name azure-devops
```

AzDO Vault internally uses `az rest`, so your identity and permissions apply.

---

## Initial Configuration

Before using the CLI, configure your Azure DevOps organizations.

### Add an organization

```bash
azdo-vault configure add \
  --name SOURCE_ORGANIZATION_ALIAS \
  --org YOUR_AZURE_DEVOPS_ORGANİZATİON_NAME
```

This maps:

```
SOURCE_ORGANIZATION_ALIAS -> https://dev.azure.com/{YOUR_AZURE_DEVOPS_ORGANİZATİON_NAME}
```

### Set default organization

```bash
azdo-vault configure default SOURCE_ORGANIZATION_ALIAS
```

### View configuration

```bash
azdo-vault configure show
```

Config is stored locally under your home directory and reused across commands.

---

## Authentication & Resource GUID

Most commands accept:

```
--ado-resource-guid
```

This is the Azure DevOps AAD resource GUID:

```
456b82ad-3146-271d-xxg56-357hq6805433
```

While some commands may work without it, **it is strongly recommended** to always provide it to avoid token resolution issues.

---

## Backup Examples

### Backup branch policies

```bash
azdo-vault backup-branch-policies \
  --source-org SOURCE_ORGANIZATION_ALIAS \
  --source-project SOURCE_PROJECT
```

### Backup build definitions

```bash
azdo-vault backup-build-definitions \
  --source-org SOURCE_ORGANIZATION_ALIAS \
  --source-project SOURCE_PROJECT \
  --definitions all \
  --ado-resource-guid ADO_RESOURCE_GUID
```

### Backup YAML pipelines

```bash
azdo-vault backup-yaml-pipelines \
  --source-org SOURCE_ORGANIZATION_ALIAS \
  --source-project SOURCE_PROJECT \
  --pipelines all \
  --ado-resource-guid ADO_RESOURCE_GUID
```

### Mirror clone repositories

```bash
azdo-vault mirror-clone \
  --org SOURCE_ORGANIZATION_ALIAS \
  --project SOURCE_PROJECT \
  --repos all
```

---

## Restore / Migration Examples

### Create repositories in target org/project

```bash
azdo-vault create-repos \
  --source-org SOURCE_ORGANIZATION_ALIAS \
  --source-project SOURCE_PROJECT \
  --target-org TARGET_ORGANIZATION_ALIAS \
  --target-project TARGET_PROJECT \
  --repos all
```

### Restore branch policies

```bash
azdo-vault create-branch-policies \
  --source-org SOURCE_ORGANIZATION_ALIAS \
  --source-project SOURCE_PROJECT \
  --target-org TARGET_ORGANIZATION_ALIAS \
  --target-project TARGET_PROJECT \
  --policies all \
  --ado-resource-guid ADO_RESOURCE_GUID
```

### Restore build definitions with queue mapping

```bash
azdo-vault create-build-definitions \
  --source-org SOURCE_ORGANIZATION_ALIAS \
  --source-project SOURCE_PROJECT \
  --target-org TARGET_ORGANIZATION_ALIAS \
  --target-project TARGET_PROJECT \
  --definitions all \
  --queue-map 'SOURCE_AGENT_POOL_NAME=TARGET_AGENT_POOL_NAME' \
  --default-queue TARGET_AGENT_POOL_NAME \
  --ado-resource-guid ADO_RESOURCE_GUID
```

### Push mirrored repositories

```bash
azdo-vault push-all-and-tags \
  --source-org SOURCE_ORGANIZATION_ALIAS \
  --source-project SOURCE_PROJECT \
  --target-org TARGET_ORGANIZATION_ALIAS \
  --target-project TARGET_PROJECT \
  --repos all
```

---

## Backup Directory Layout

```
~/azdo-vaults/
└── SOURCE_ORGANIZATION_ALIAS/
    └── SOURCE_PROJECT/
        ├── repos/
        ├── branch-policies/
        ├── build-definitions/
        ├── release-definitions/
        ├── yaml-pipelines/
        ├── service-connections/
        ├── task-groups/
        ├── variable-groups/
        ├── artifacts/
        │   └── feeds/
        └── wikis/
```

This structure is intentionally human-readable and version-control friendly.

---

## Idempotency & Safety

AzDO Vault is designed to be **safe to re-run**:

* Existing resources are detected and skipped where possible
* Server-managed fields are stripped before restore
* Mappings (queues, identities, repos) are resolved dynamically

Some operations (like branch policies) may skip items if dependencies cannot be resolved.

---

## License

MIT License.  
See the `LICENSE` file for full text.

---

## Disclaimer

This tool is **not affiliated with Microsoft**.
Use with care and test migrations in non-production environments first.
