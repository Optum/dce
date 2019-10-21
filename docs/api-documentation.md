# ${namespace}
## Version: 1.0

### Security
**sigv4**  

|apiKey|*API Key*|
|---|---|
|Name|Authorization|
|In|header|
|x-amazon-apigateway-authtype|awsSigv4|

### /accounts

#### GET
##### Summary:

Lists accounts

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A list of accounts | [ [account](#account) ] |
| 403 | Unauthorized |  |

##### Security

| Security Schema | Scopes |
| --- | --- |
| sigv4 | |

#### POST
##### Summary:

Add an AWS Account to the Redbox account pool

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| account | body | Account creation parameters | No | object |

##### Responses

| Code | Description |
| ---- | ----------- |
| 201 |  |
| 403 | Failed to authenticate request |

##### Security

| Security Schema | Scopes |
| --- | --- |
| sigv4 | |

### /accounts/{id}

#### GET
##### Summary:

Get a specific account by an account ID

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| id | path | accountId for lease | Yes | string |

##### Responses

| Code | Description |
| ---- | ----------- |
| 200 |  |
| 403 | Failed to retrieve account |

##### Security

| Security Schema | Scopes |
| --- | --- |
| sigv4 | |

#### DELETE
##### Summary:

Delete an account by ID.

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| id | path | The ID of the account to be deleted. | Yes | string |

##### Responses

| Code | Description |
| ---- | ----------- |
| 204 | The account has been successfully deleted. |
| 403 | Unauthorized. |
| 404 | No account found for the given ID. |
| 409 | The account is unable to be deleted. |

##### Security

| Security Schema | Scopes |
| --- | --- |
| sigv4 | |

### /leases

#### POST
##### Summary:

Creates a new lease.

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| lease | body | The owner of the lease | No | object |

##### Responses

| Code | Description |
| ---- | ----------- |
| 201 |  |
| 403 | Failed to authenticate request |

##### Security

| Security Schema | Scopes |
| --- | --- |
| sigv4 | |

#### DELETE
##### Summary:

Removes a lease.

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| lease | body | The owner of the lease | No | object |

##### Responses

| Code | Description |
| ---- | ----------- |
| 201 | Lease successfully removed |
| 403 | Failed to authenticate request |

##### Security

| Security Schema | Scopes |
| --- | --- |
| sigv4 | |

#### GET
##### Summary:

Get leases

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| principalId | query | Principal ID of the leases. | No | string |
| accountId | query | Account ID of the leases. | No | string |
| status | query | Status of the leases. | No | string |
| nextPrincipalId | query | Principal ID with which to begin the scan operation. This is used to traverse through paginated results. | No | string |
| nextAccountId | query | Account ID with which to begin the scan operation. This is used to traverse through paginated results. | No | string |
| limit | query | The maximum number of leases to evaluate (not necessarily the number of matching leases). If there is another page, the URL for page will be in the response Link header. | No | integer |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | OK | [ [lease](#lease) ] |
| 403 | Failed to authenticate request |  |

##### Security

| Security Schema | Scopes |
| --- | --- |
| sigv4 | |

### /leases/{id}

#### GET
##### Summary:

Get a lease by Id

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| id | path | Id for lease | Yes | string |

##### Responses

| Code | Description |
| ---- | ----------- |
| 200 |  |
| 403 | Failed to retrieve lease |

##### Security

| Security Schema | Scopes |
| --- | --- |
| sigv4 | |

### /usage

#### GET
##### Summary:

Get usage records by date range

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ---- |
| startDate | path | start date of the usage | Yes | number |
| endDate | path | end date of the usage | Yes | number |

##### Responses

| Code | Description |
| ---- | ----------- |
| 200 |  |
| 403 | Failed to authenticate request |

##### Security

| Security Schema | Scopes |
| --- | --- |
| sigv4 | |

### Models


#### lease

Lease Details

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| id | string | Lease ID | No |
| principalId | string | principalId of the lease to get | No |
| accountId | string | accountId of the AWS account | No |
| leaseStatus | [leaseStatus](#leasestatus) |  | No |
| createdOn | number | creation date in epoch seconds | No |
| lastModifiedOn | number | date last modified in epoch seconds | No |
| budgetAmount | number | budget amount | No |
| budgetCurrency | string | budget currency | No |
| budgetNotificationEmails | [ string ] | budget notification emails | No |
| leaseStatusModifiedOn | number | date lease status was last modified in epoch seconds | No |
| requestedLeaseEnd | number | date lease should expire in epoch seconds | No |

#### account

Account Details

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| id | string | AWS Account ID | No |
| accountStatus | [accountStatus](#accountstatus) |  | No |
| adminRoleArn | string | ARN for an IAM role within this AWS account. The Redbox master account will assume this IAM role to execute operations within this AWS account. This IAM role is configured by the client, and must be configured with [a Trust Relationship with the Redbox master account.](/https://docs.aws.amazon.com/IAM/latest/UserGuide/tutorial_cross-account-with-roles.html) | No |
| principalRoleArn | string | ARN for an IAM role within this AWS account. This role is created by the Redbox master account, and may be assumed by principals to login to their AWS Redbox account. | No |
| principalPolicyHash | string | The S3 object ETag used to apply the Principal IAM Policy within this AWS account.  This policy is created by the Redbox master account, and is assumed by people with access to principalRoleArn. | No |
| lastModifiedOn | integer | Epoch timestamp, when account record was last modified | No |
| createdOn | integer | Epoch timestamp, when account record was created | No |
| metadata | object | Any organization specific data pertaining to the account that needs to be persisted | No |

#### accountStatus

Status of the Account.
"Ready": The account is clean and ready for lease
"NotReady": The account is in "dirty" state, and needs to be reset before it may be leased.
"Leased": The account is leased to a principal


| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| accountStatus | string | Status of the Account. "Ready": The account is clean and ready for lease "NotReady": The account is in "dirty" state, and needs to be reset before it may be leased. "Leased": The account is leased to a principal  |  |

#### leaseStatus

Status of the Lease.
"Active": The principal is leased and has access to the account
"Decommissioned": The principal was previously leased to the account, but now is not.
"FinanceLock": The principal is leased to the account, but has hit a budget threshold, and is locked out of the account.
"ResetLock": The principal is leased to the account, but the account is being reset. The principal's access is temporarily revoked, and will be given back after the reset process is complete.
"ResetFinanceLock": The principal is leased to the account, but has been locked out for hitting a budget threshold. Additionally, the account is being reset. After reset, the principal's access will _not_ be restored, and the LeaseStatus will be set back to `ResetLock`.


| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| leaseStatus | string | Status of the Lease. "Active": The principal is leased and has access to the account "Decommissioned": The principal was previously leased to the account, but now is not. "FinanceLock": The principal is leased to the account, but has hit a budget threshold, and is locked out of the account. "ResetLock": The principal is leased to the account, but the account is being reset. The principal's access is temporarily revoked, and will be given back after the reset process is complete. "ResetFinanceLock": The principal is leased to the account, but has been locked out for hitting a budget threshold. Additionally, the account is being reset. After reset, the principal's access will _not_ be restored, and the LeaseStatus will be set back to `ResetLock`.  |  |

#### usage

usage cost of the aws account from start date to end date

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| principalId | string | principalId of the user who owns the lease of the AWS account | No |
| accountId | string | accountId of the AWS account | No |
| startDate | number | usage start date as Epoch Timestamp | No |
| endDate | number | usage end date as Epoch Timestamp | No |
| costAmount | number | usage cost Amount of AWS account for given period | No |
| costCurrency | string | usage cost currency | No |
| timeToLive | number | ttl attribute as Epoch Timestamp | No |