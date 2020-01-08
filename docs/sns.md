# SNS Lifecycle Events

The DCE master account publishes messages to a number of SNS topics, to indicate lifecycle events. This allows DCE system administrators to customize their implementation of DCE by subscribing and reacting to these events. 

For example, you could setup an _auto-renewal_ system by listening to the `lease-removed` SNS topic, and triggering a Lambda that recreates the lease as soon as it expires.

See the `Extending Terraform Configuration <terraform.html#extending-the-terraform-configuration>`_ documentation, for an example of using Terraform to subscribe to DCE SNS topics  


## account-created

An account was added to the account pool


This SNS topic ARN is provided as `a Terraform output <terraform.html#deploy-with-terraform>`_:

```
terraform output account_created_topic_arn
```

#### Payload

This message includes a payload as JSON, with the following fields:

| Field          | Type                             | Description                                                                                                 |
| -------------- | -------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| id             | string                           | AWS Account ID                                                                                              |
| accountStatus  | "Ready", "NotReady", "Orphaned", or "Leased" | Account status                                                                                              |
| adminRoleArn   | string                           | ARN for the IAM role used by the DCE master account to manage the account                                |
| lastModifiedOn | int                              | Last modified timestamp                                                                                     |
| createdOn      | int                              | Last modified timestamp                                                                                     |
| metadata       | JSON object                      | Metadata field contains any organization specific data pertaining to the account that needs to be persisted |

Example:

```json
{
  "id": "1234567890",
  "accountStatus": "NotReady",
  "adminRoleArn": "arn:aws:iam::1234567890123:role/adminRole",
  "principalRoleArn": "arn:aws:iam::1234567890123:role/DCEPrincipal",
  "principalPolicyHash": "\"d41d8cd98f00b204e9800998ecf8427e-38\"",
  "createdOn": 1560306008,
  "lastModifiedOn": 1560306008,
  "metadata": {}
}
```


## account-deleted

An account was deleted from the account pool

This SNS topic ARN is provided as `a Terraform output <terraform.html#deploy-with-terraform>`_:

```
terraform output account_deleted_topic_arn
```


#### Payload

This message includes a payload as JSON, with the following fields:

| Field          | Type                             | Description                                                                                                 |
| -------------- | -------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| id             | string                           | AWS Account ID                                                                                              |
| accountStatus  | "Ready", "NotReady", "Orphaned", or "Leased" | Account status                                                                                              |
| adminRoleArn   | string                           | ARN for the IAM role used by the DCE master account to manage the account                                |
| lastModifiedOn | int                              | Last modified timestamp                                                                                     |
| createdOn      | int                              | Last modified timestamp                                                                                     |
| metadata       | JSON object                      | Metadata field contains any organization specific data pertaining to the account that needs to be persisted |


Example:

```json
{
  "id": "1234567890",
  "accountStatus": "NotReady",
  "adminRoleArn": "arn:aws:iam::1234567890123:role/adminRole",
  "principalRoleArn": "arn:aws:iam::1234567890123:role/DCEPrincipal",
  "principalPolicyHash": "\"d41d8cd98f00b204e9800998ecf8427e-38\"",
  "createdOn": 1560306008,
  "lastModifiedOn": 1560306008,
  "metadata": {}
}
```

## lease-added

Triggered when a lease is created.

This SNS topic ARN is provided as `a Terraform output <terraform.html#deploy-with-terraform>`_:

```
terraform output lease_added_topic_arn
```

#### Payload

This message includes a payload as JSON, with the following fields:

| Field           | Type    | Description                                         |
| --------------- | ------- | --------------------------------------------------- |
| accountId       | string  | AWS Account ID                                      |
| principalId     | string  | ID of the principal user, associated with the lease |
| leaseStatus     | string  | Status of the lease.                                |
| createdOn       | integer | Timestamp (epoch) of creation                       |
| lastModifiedOn  | integer | Timestamp (epoch) of last modification              |
| leaseModifiedOn | integer | Timestamp (epoch) of lease status modification      |
| expiresOn | integer | Timestamp (epoch) when the lease will expire |

Example:

```json
{
  "accountId": "1234567890",
  "principalId": "jdoe17",
  "leaseStatus": "Active",
  "createdOn": 1560306008,
  "lastModifiedOn": 1560306008,
  "leaseStatusModifiedOn": 1560306008,
  "expiresOn": 1560306008
}
```

## lease-removed

Triggered when a lease is deleted.

This SNS topic ARN is provided as `a Terraform output <terraform.html#deploy-with-terraform>`_:

```
terraform output lease_removed_topic_arn
```

#### Payload

This message includes a payload as JSON, with the following fields:

| Field                 | Type    | Description                                         |
| --------------------- | ------- | --------------------------------------------------- |
| accountId             | string  | AWS Account ID                                      |
| principalId           | string  | ID of the principal user associated with the lease  |
| leaseStatus           | string  | Status of the lease.                                |
| createdOn             | integer | Timestamp (epoch) of creation                       |
| lastModifiedOn        | integer | Timestamp (epoch) of last modification              |
| leaseStatusModifiedOn | integer | Timestamp (epoch) of last lease status modification |
| expiresOn | integer | Timestamp (epoch) when the lease will expire |

Example:

```json
{
  "accountId": "1234567890",
  "principalId": "jdoe17",
  "leaseStatus": "Decommissioned",
  "createdOn": 1560306008,
  "lastModifiedOn": 1560306008,
  "leaseStatusModifiedOn": 1560306008,
  "expiresOn": 1560306008
}
```
