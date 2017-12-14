#!/bin/sh

kubectl delete deploy --all
kubectl delete customresourcedefinition --all
kubectl delete ingress --all
kubectl delete po --all
kubectl delete svc --all
kubectl delete secret k8s-certs kubeconfig
kubectl apply -f artifacts/