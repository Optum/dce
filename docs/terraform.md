## Deploying DCE with Terraform

The AWS infrastructure for the DCE master account is defined as a Terraform module within the [github.com/Optum/dce](https://github.com/Optum/dce) repo. This infrastructure may be deployed using the [Terraform CLI](https://www.terraform.io/docs/commands/index.html):

```
cd modules
terraform init
terraform apply
``` 

See [terraform.io](https://www.terraform.io/) for more information on using Terraform.\

After the Terraform deployment is complete, you will need to build and deploy the application code to AWS:

```
make deploy
``` 

Alternatively, you can download the build artifacts from [a Github release](https://github.com/Optum/dce/releases), and deploy them directly.
Both the `deploy.sh` and `build_artifacts.zip` are supplied with the github release:

```
cd modules
namespace=$(terraform output namespace)
artifacts_bucket=$(terraform output artifacts_bucket_name)
deploy.sh build_artifacts.zip ${namespace} ${artifacts_bucket}
```

## Configuring Terraform Variables

The DCE Terraform module accepts a number of configuration variables to tweak the behavior of the DCE deployment. These variables can be provided to the `terraform apply` CLI command, or configured in a `tfvars` file.
 
 For example:
 
```
terraform apply \
    -var namespace=nonprod \
    -var check_budget_enabled=false \
    -var-file my-dce.tfvars
```
 
See [Terraform documentation for details on configuring input variables](https://www.terraform.io/docs/configuration/variables.html).

See [/modules/variables.tf](https://github.com/Optum/dce/blob/master/modules/variables.tf) for a full list of configurable Terraform variables.

## Accessing Terraform Outputs

The DCE Terraform module outputs a number of parameters, which may be useful for interacting with the configured resources. For example, the `api_url` output provides the base url for your DCE API Gateway endpoint.

Use the [`terraform output`](https://www.terraform.io/docs/commands/output.html) CLI command to access outputs.

```bash
cd modules
terraform output api_url
```

For a full list of available outputs, see [/modules/outputs.tf](https://github.com/Optum/dce/blob/master/modules/outputs.tf)
 
 
## Extending the Terraform Configuration

You may want to extend the DCE Terraform configuration with our own infrastructure. For example, you may want to subscribe your own Lambda to DCE [SNS Lifecycle Events](sns.md). 

To do this, pull in the DCE Terraform module as a submodule from within your own Terraform configuration:

```hcl
# Load DCE as a Terraform submodule
module "dce" {
  source = "github.com/Optum/dce//modules"
  # Optionally, configure additional input variables
  namespace= "nonprod"
  check_budget_enabled = false
}

# Reference DCE module outputs as needed
# For example, here we'll subscribe to the "lease-added" SNS topic
resource "aws_sns_topic_subscription" "assign_topic_lambda" {
  topic_arn = module.dce.lease_added_topic_arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.my_fn.arn
}
resource "aws_lambda_permission" "assign_sns" {
  statement_id  = "AllowExecutionFromSNS"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.my_fn.name
  principal     = "sns.amazonaws.com"
  source_arn    = module.dce.lease_added_topic_arn
}
```