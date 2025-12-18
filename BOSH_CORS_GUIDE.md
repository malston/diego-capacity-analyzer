# Adding CORS to BOSH Manifest: Step-by-Step Guide

## File: bosh-cf-85da7fd88e99806e5d08.yml

## Step 1: Find the Gorouter Job

In your **Small Footprint** deployment, gorouter runs inside the `router` or `control` instance group.

### Search for Gorouter

```bash
# Find the gorouter job
grep -n "name: gorouter" bosh-cf-85da7fd88e99806e5d08.yml

# Or search for the router instance group
grep -n "- name: router" bosh-cf-85da7fd88e99806e5d08.yml
grep -n "- name: control" bosh-cf-85da7fd88e99806e5d08.yml
```

In **Small Footprint**, it's likely in one of these:
- `instance_groups: - name: router`
- `instance_groups: - name: control`

## Step 2: Locate the Gorouter Properties Section

Look for this structure:

```yaml
instance_groups:
- name: router  # or "control" in Small Footprint
  jobs:
  - name: gorouter
    properties:
      router:
        # This is where we add CORS
```

## Step 3: Add CORS Configuration

**Add this property** under `properties.router:`:

```yaml
        cors_allowed_origins:
        - "http://localhost:3000"
        - "http://127.0.0.1:3000"
```

### Full Example Context

Here's what it should look like in context:

```yaml
instance_groups:
- name: router
  jobs:
  - name: gorouter
    properties:
      router:
        # Existing properties (don't remove these!)
        status:
          port: 8080
          user: router_status
          password: ((router_status_password))
        
        # ADD THIS NEW SECTION
        cors_allowed_origins:
        - "http://localhost:3000"
        - "http://127.0.0.1:3000"
        
        # Other existing properties continue below...
        drain_wait: 20
        enable_ssl: true
        # etc...
```

## Step 4: Exact Commands to Find and Edit

### 1. Find the line number where gorouter properties start

```bash
grep -n "name: gorouter" bosh-cf-85da7fd88e99806e5d08.yml
```

This will show something like:
```
1234:  - name: gorouter
```

### 2. View the gorouter section

```bash
# Replace XXXX with the line number from step 1
sed -n 'XXXX,+100p' bosh-cf-85da7fd88e99806e5d08.yml
```

This shows you 100 lines starting from the gorouter job, so you can see the structure.

### 3. Find where to insert CORS

Look for the `properties:` and then `router:` section. CORS should go directly under `router:`.

## Step 5: Make a Backup First!

```bash
cp bosh-cf-85da7fd88e99806e5d08.yml bosh-cf-85da7fd88e99806e5d08.yml.backup
```

## Step 6: Edit the File

```bash
# Option 1: Use vim
vim bosh-cf-85da7fd88e99806e5d08.yml

# Option 2: Use your preferred editor
code bosh-cf-85da7fd88e99806e5d08.yml  # VS Code
nano bosh-cf-85da7fd88e99806e5d08.yml  # Nano
```

**When editing:**
1. Find the gorouter job (search for `/gorouter`)
2. Find the `properties:` section under gorouter
3. Find the `router:` section under properties
4. Add the CORS lines (with proper indentation!)

### Critical: Indentation Matters!

YAML is indentation-sensitive. Make sure:
- `cors_allowed_origins:` aligns with other properties under `router:`
- The list items (`- "http://...`) are indented 2 more spaces

## Step 7: Verify Your Changes

After editing, check the syntax:

```bash
# Check for CORS settings
grep -A 3 "cors_allowed_origins" bosh-cf-85da7fd88e99806e5d08.yml
```

Should show:
```yaml
        cors_allowed_origins:
        - "http://localhost:3000"
        - "http://127.0.0.1:3000"
```

## Step 8: Deploy the Changes

```bash
# Get your deployment name
bosh deployments

# Deploy (replace cf-XXXXX with your actual deployment name)
bosh -d cf-85da7fd88e99806e5d08 deploy bosh-cf-85da7fd88e99806e5d08.yml
```

## Step 9: Verify After Deployment

```bash
# SSH to router VM
bosh -d cf-85da7fd88e99806e5d08 ssh router/0

# Or if using control instance group:
bosh -d cf-85da7fd88e99806e5d08 ssh control/0

# Check gorouter config
sudo cat /var/vcap/jobs/gorouter/config/gorouter.yml | grep -A 3 cors
```

Should show:
```yaml
cors_allowed_origins:
- http://localhost:3000
- http://127.0.0.1:3000
```

## Troubleshooting

### Can't Find Gorouter?

In Small Footprint, try searching for all instance groups:

```bash
grep "^- name:" bosh-cf-85da7fd88e99806e5d08.yml | grep -E "router|control|compute"
```

### YAML Syntax Error When Deploying?

Check indentation carefully:

```bash
# Install yamllint if you don't have it
brew install yamllint  # or: pip install yamllint

# Validate syntax
yamllint bosh-cf-85da7fd88e99806e5d08.yml
```

### Deployment Fails?

```bash
# Check BOSH task logs
bosh tasks --recent

# Get the task ID that failed and view logs
bosh task TASK_ID --debug
```

---

## Quick Reference

**Backup:**
```bash
cp bosh-cf-85da7fd88e99806e5d08.yml bosh-cf-85da7fd88e99806e5d08.yml.backup
```

**Edit:**
```bash
vim bosh-cf-85da7fd88e99806e5d08.yml
# Search: /gorouter
# Add CORS under properties.router:
```

**Deploy:**
```bash
bosh -d cf-85da7fd88e99806e5d08 deploy bosh-cf-85da7fd88e99806e5d08.yml
```

**Verify:**
```bash
bosh -d cf-85da7fd88e99806e5d08 ssh router/0
sudo cat /var/vcap/jobs/gorouter/config/gorouter.yml | grep cors
```

---

Need help? Let me know what you see when you search for gorouter!
