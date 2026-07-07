# FinLedger Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Pod REJECTED on create | Pod Security `restricted` violation (root/priv) | harden securityContext; the manifests already comply |
| transaction-api CrashLoop on start | DB not ready / wrong creds | check Postgres Ready; verify DB_PASSWORD secret; JVM needs ~30-40s (startup probe) |
| `409 insufficient_funds` | source account lacks balance (working as intended) | fund from account 0, or lower the amount |
| reconciliation Job fails | ledger out of balance (real issue) or DB unreachable | check the imbalance amount in logs; investigate the ledger |
| NetworkPolicy not blocking | default CNI doesn't enforce | install Calico/Cilium |
| DNS broken after default-deny | forgot DNS egress allow | `allow-dns-egress` policy is included — ensure it's applied |
| auditor can read secrets | RBAC too broad | the Role deliberately excludes secrets; verify with `auth can-i` |
| HPA `<unknown>` | metrics-server missing | install with `--kubelet-insecure-tls` on KIND |
| EKS: secrets not encrypted | cluster secretsEncryption not enabled | recreate/enable KMS envelope encryption |
| EKS: pod can't read Secrets Manager | IRSA misconfigured | verify OIDC, SA annotation, role policy scoped to the secret |

**Correctness check:** the ledger must always sum to zero. If reconciliation fails, a debit
exists without its matching credit — investigate the transfer that broke atomicity (should be
impossible given @Transactional, but the job is the safety net).
