# Talos Cockpit

# Prerequisites

## Talos Cluster
Follow those [steps](https://www.talos.dev/v1.8/advanced/talos-api-access-from-k8s/) to enable Talos API Access From Kubernetes

At least:
* [Enable the feature](https://www.talos.dev/v1.8/advanced/talos-api-access-from-k8s/#enabling-the-feature)

> [!WARNING]  
    Talos cockpit needs ```os:admin``` to be allowed in the feature to manage Operating System and K8S Updates 

## Cockpit Config

There are 2 ways for you to configure talos-cockpit file or Env

### Using **config.yml** 

You'll have to mount the file into /app/

> [!NOTE] config.yml
> ```yaml
> #Global configurations
> global:
>   debug: false
>
> # schedule configurations
> schedule:
>   sync_members: 15 # in Minutes
>
> # Talosctl configurations
> talosctl:
>   endpoint: "localhost" 
> 
> # Database credentials (unused)
> database:
>   user: "admin"
>   pass: "super-pedro-1982"
> ```

### Env vars

Define Env vas in you talos-cockpit pod

| Variable Name | Type | Description | Default value |
| ------------------ | ---- | ----------- | ------------- |
| COCKPIT_DEBUG | Boolean | Enable log/runtime debug | False |
| COCKPIT_SCHED_SYNC | Int | Sync and updates each X Minutes | 5 |
| COCKPIT_SCHED_SYS_UPGRADE | Int | Sync and updates each X Minutes | 10 |
| **TALOS_API_ENDPOINT** | String | Endpoint API used by talos-cockpit | |
| DB_USERNAME | String | NOT USED | |
| DB_PASSWORD | String | NOT USED | |

***Vars Required***

# Deploy and enjoy

Use the example above to deploy a pod in the ````namespace allowed```` to use Talos API.

Replace the **TALOS_API_ENDPOINT** env var by **your controllers vIP** or the IP/Name of one of your controllers

> [!WARNING]  
    TALOS_API_ENDPOINT value should have been declared in Talos SANs

> [!NOTE] cockpit-deployment.yaml
> ```yaml
> apiVersion: apps/v1
> kind: Deployment
> metadata:
>   creationTimestamp: null
>   name: talos-cockpit
> spec:
>   selector:
>     matchLabels:
>       app: talos-cockpit
>   strategy: {}
>   template:
>     metadata:
>       creationTimestamp: null
>       labels:
>         app: talos-cockpit
>     spec:
>       containers:
>       - image: mstrohl/talos-cockpit:0.0.1
>         imagePullPolicy: Always
>         name: talos-cockpit
>         resources: {}
>         volumeMounts:
>         - mountPath: /var/run/secrets/talos.dev
>           name: talos-secrets
>         env:
>           - name: TALOS_API_ENDPOINT
>             value: "10.0.0.15"
>       volumes:
>       - name: talos-secrets
>         secret:
>           secretName: talos-cockpit-talos-secrets
> status: {}
> ---
> apiVersion: talos.dev/v1alpha1
> kind: ServiceAccount
> metadata:
>     name: talos-cockpit-talos-secrets
> spec:
>     roles:
>         - os:admin
>  ```