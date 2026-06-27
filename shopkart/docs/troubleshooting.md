# ShopKart Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Pods `ImagePullBackOff` (KIND) | images not loaded into KIND | run `manifests/kind/build-and-load.sh` |
| Pods `ImagePullBackOff` (EKS) | wrong ECR URI / node role lacks ECR pull | check image URI; add `AmazonEC2ContainerRegistryReadOnly` to node role |
| Service unreachable, pods healthy | selector ≠ labels → empty endpoints | `kubectl -n shopkart get endpoints <svc>`; align selector |
| Pod `Running` but no traffic | readiness probe failing (DB/Redis down) | `kubectl -n shopkart logs <pod>`; check DB/Redis reachable |
| 502 from gateway | backend down or selector wrong | check the target service's pods + endpoints |
| Catalog/orders/users crash on start | DB not reachable / wrong DB_NAME | verify Postgres ready; check `DB_NAME` env + that the DB exists |
| HPA `<unknown>` | metrics-server missing | install it (`--kubelet-insecure-tls` on KIND) |
| Ingress 404 | path rules / controller missing | `kubectl get ingress`; ensure controller installed |
| NetworkPolicy not blocking | non-enforcing CNI | use Calico/Cilium (default KIND CNI ignores NetPol) |
| EKS: Ingress, no ALB | LB Controller not installed/permitted | install the AWS Load Balancer Controller with its IRSA role |
| Pod can't read Secrets Manager (EKS) | IRSA misconfigured | verify OIDC, SA annotation, pod uses the SA, role policy |

**Master move:** `kubectl -n shopkart get pods` → `describe`/`logs -p` for the failing one → `get endpoints <svc>` for networking. The two-layer split (platform via describe/events, app via logs) diagnoses almost everything.
