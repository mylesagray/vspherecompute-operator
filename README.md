# Bridging Infrastructure and Applications (BIA) - vSphereCompute

A very rough K8s operator and CRD scheme for creating and deleting vSphere compute types (VMs).

## Setup

### Config

*Requires Kubernetes 1.13+*

Update the `value` field for the following in [deploy/operator.yaml](deploy/operator.yaml) to reflect your environment:

```yaml
- name: GOVC_URL
  value: "https://username:password@vCenterFQDNorIP"
```

Import the vSphereCompute CRDs

```sh
kubectl create -f deploy/crds/bia_v1alpha1_vspherecompute_crd.yaml
```

Import the operator

```sh
kubectl create -f deploy/
```

## Usage

Create a `vSphereCompute` object reflecting the VM(s) you want created, an example can be found at [deploy/crds/bia_v1alpha1_vspherecompute_cr.yaml](deploy/crds/bia_v1alpha1_vspherecompute_cr.yaml)

Import the `vSphereCompute` object into K8s:

```sh
kubectl create -f deploy/crds/bia_v1alpha1_vspherecompute_cr.yaml
```

A VM will be spun up on the environment as specified in [deploy/operator.yaml](deploy/operator.yaml).

Get the current BIA deployed and managed VMs
```sh
$ kubectl get vc
NAME          VMNAME        STATUS      CPUS   MEMORY   IP    HOST
hello-world   hello-world   poweredOn   2      2048           host-28
```

## Cleanup

```sh
kubectl delete -f deploy/crds/
kubectl delete -f deploy/
```

## Known Issues

* Lots of hard coded values. Tons.
* Uses a hammer-solution of govc instead of govmomi to interact with VC via os.exec
* Doesn't use the VMware VCP settings set on the cluster - relies on it's own ENV vars
* Only CPU, Memory and VMName implemented from CR spec so far.
* Doesnt resolve VC moref-ids to human-relatable names.
* Does not update VMs with patched changes.
* Likely uses the wrong call for deleting VMs.
