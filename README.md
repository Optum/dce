# Disposable Cloud Environment<sup>TM</sup>

> **DCE<sup>TM</sup> is your playground in the cloud**

DCE helps you quickly and safely explore the public cloud by managing temporary AWS accounts.

Common use cases for a public cloud account include:
- Developing, testing, or operating cloud networks and applications
- Improving infrastructure utilization with autoscaling
- Leveraging cloud-native developer tooling
- Exploring data with analytical and machine learning services
- And much [more](https://aws.amazon.com/)!

DCE users can "lease" an AWS account for a defined period of time and with a limited budget.

At the end of the lease, or if the lease's budget is reached, the account is wiped clean and returned to the account pool so it may be leased again.

## Getting Started & Documentation

[Disposable Cloud Environment (DCE)’s documentation!](https://dce.readthedocs.io/en/latest/).

## DCE CLI

The easiest way to get started with DCE is with the DCE CLI:

[github.com/Optum/dce-cli](https://github.com/Optum/dce-cli)

```bash
# Deploy DCE
dce system deploy

# Add an account to the pool
dce accounts add \
    --account-id 123456789012 \
    --admin-role-arn arn:aws:iam::123456789012:role/OrganizationAccountAccessRole

# Lease an account
dce leases create \
    --principal-id jdoe@example.com \
    --budget-amount 100 --budget-currency USD

# Login to your account
dce leases login <lease-id>
```

## Contributing to DCE

DCE was born at Optum, but belongs to the community. Improve your cloud experience and [open a PR](https://github.com/Optum/dce/pulls).

[Contributor Guidelines](./CONTRIBUTING.md)


## License

[Apache License v2.0](./LICENSE)
