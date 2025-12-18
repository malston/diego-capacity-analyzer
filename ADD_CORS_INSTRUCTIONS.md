# Instructions for Adding CORS to cf.yml

## 1. Open cf.yml in your editor

```bash
vim /Users/markalston/workspace/diego-capacity-analyzer/cf.yml
```

## 2. Find the gorouter properties section

Search for `.properties.gorouter_` - you'll find properties like:
- `.properties.gorouter_customize_metrics_reporting:`
- `.properties.gorouter_ssl_ciphers:`

## 3. Add the CORS property

**Add this ANYWHERE in the `product-properties:` section** (good spot is near other gorouter properties):

```yaml
  .properties.gorouter_cors_allowed_origins:
    value:
    - http://localhost:3000
    - http://127.0.0.1:3000
```

**Example placement** (after `.properties.gorouter_ssl_ciphers:`):

```yaml
  .properties.gorouter_ssl_ciphers:
    value: TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384
  .properties.gorouter_cors_allowed_origins:
    value:
    - http://localhost:3000
    - http://127.0.0.1:3000
  .properties.isolated_routing:
    selected_option: accept_all
```

## 4. Apply the config back to Ops Manager

```bash
# From your Ops Manager host or jump box
om --target https://YOUR-OPSMGR-URL \
   --username admin \
   --password YOUR-PASSWORD \
   --skip-ssl-validation \
   configure-product \
   --product-name cf \
   --config cf.yml
```

## 5. Apply Changes in Ops Manager

1. Go to Ops Manager dashboard
2. Click "Review Pending Changes"
3. Select only TAS tile
4. Click "Apply Changes"

---

## Alternative: Direct BOSH Approach

If you prefer to skip Ops Manager entirely and go straight to BOSH:

### 1. Get the actual BOSH manifest

```bash
bosh -d cf-* manifest > bosh-manifest.yml
```

### 2. Edit the BOSH manifest

Find the `router` instance group and add CORS to gorouter properties:

```yaml
instance_groups:
- name: router  # Might be "compute" in Small Footprint
  jobs:
  - name: gorouter
    properties:
      router:
        cors_allowed_origins:
        - "http://localhost:3000"
        - "http://127.0.0.1:3000"
```

### 3. Deploy

```bash
bosh -d cf-* deploy bosh-manifest.yml
```

This bypasses Ops Manager entirely but achieves the same result.

---

## Which Method Should You Use?

**If you're comfortable with BOSH** (you are):
→ Use the Direct BOSH Approach (faster, more direct)

**If you want it tracked in Ops Manager**:
→ Use the Ops Manager config approach

---

## Next Steps

Let me know which approach you want to take and I'll help you through it!
