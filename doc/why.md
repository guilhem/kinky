# ![Why?](assets/butwhy.gif)

Spawning a kubernetes cluster may seems difficult.
You have to spawn many components and a database without any orchestrator... to spawn an orchestrator.
It's a chicken / egg problem.

Many tools exists [kubeadm](https://kubernetes.io/docs/setup/independent/install-kubeadm/), [kubespray](https://github.com/kubernetes-incubator/kubespray), [RKE](https://github.com/rancher/rke)...
All are really good tools but needs knowledge and practice.

What we want to achieve is to create an easy KaaS (Kubernetes as a Service).
It can be used by company with their own infrastructure.

It should be really useful when you are organized as feature teams.
* A team manage the KaaS cluster.
* Others teams manage their servers but make them join a managed k8s cluster
