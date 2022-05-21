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

In order to run this configuration setup, you will first need to activate an AWS account. Like the other cloud platoforms, you can set one up for free if you want to experiment, however, keep in mind all three will require a credit card on file and will charge a small fee for some of the resources.  

## Creating a Cloud Account  
   Create an AWS account by browsing to: [AWS Web Console](https://aws.amazon.com)  
   Click on **Create a Free Account**  
   [](insertfileimage)  
   If you don't have a cuurent account, complete the signup form to create a free tier account  
   [](insertfileimage)
   ***You will need to have access to the root username and password***  
   

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
