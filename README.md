# Argo chaos mesh plugin
## Installation

```bash
git clone git@github.com:xlgao-zju/argo-chaos-mesh-plugin.git
cd argo-chaos-mesh-plugin
kubectl apply -f ./deploy
```

## Run the demon

```bash
# run a pod as a experiment target
kubectl run nginx --image=nginx
# run argo workflow which uses the chaos mesh plugin
kubectl create -f .doc/demo.yaml
```