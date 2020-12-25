# EKS-Fargate-Controller
The eks-fargate-controller is a k8s operator that allows you to create and manage eks-fargate profiles
dynamically using the same k8s control plane. You declaratively define what your desired fargate-profile 
will look like and simply create it using `kubectl apply -f eks-fargate-profile.yaml`.

## Installation
