###############################################################################################################################
# This script may be used as an example for deleting resources deployed by dce-cli previous to dce-cli version 0.4.0          #
# It uses AWS Resource Groups to query for resources tagged AppName=DCE. After deleting these resources and the               #
# resource group, it deletes additional resources using substring matching on arns and resource names for the                 #
# namespace provided as the second parameter to the script.                                                                   #
#                                                                                                                             #
# It is important to exercise caution any time you are deleting AWS resources in bulk. Please either read this script         #
# thoroughly or use it as a guide if you have concerns about accidentally deleting other AWS resources.                       #
###############################################################################################################################

##!/usr/bin/env bash
#
#REGION=$1
#NAMESPACE=$2
#ACCOUNT=$(aws sts get-caller-identity --output text --query '{Account:Account}')
#RESOURCE_GROUP_NAME=DELETEDCE
#
#function delete_resource {
#  RESOURCE_TYPE=$1
#  RESOURCE_ARN=$2
#
#  echo "Attempting to delete $RESOURCE_TYPE - $RESOURCE_ARN"
#
#  ARN_SUFFIX=$(echo $RESOURCE_ARN | sed -E 's/^.*:([^:]+)$/\1/')
#  case $RESOURCE_TYPE in
#
#  "AWS::CloudWatch::Alarm")
#    aws cloudwatch delete-alarms --alarm-names $ARN_SUFFIX
#    ;;
#
#  "AWS::CodeBuild::Project")
#    PROJECT_NAME=$(echo $ARN_SUFFIX | sed -E 's/^.*\///')
#    aws codebuild delete-project --name $PROJECT_NAME
#    ;;
#
#  "AWS::DynamoDB::Table")
#    TABLE_NAME=$(echo $ARN_SUFFIX | sed -E 's/^.*\///')
#    aws dynamodb delete-table --table-name $TABLE_NAME
#    ;;
#
#  "AWS::Lambda::Function")
#    aws lambda delete-function --function-name $ARN_SUFFIX
#    ;;
#
#  "AWS::Lambda::EventSourceMapping")
#    aws lambda delete-event-source-mapping --uuid $ARN_SUFFIX
#    ;;
#
#  "AWS::SNS::Topic")
#    TOPIC_SUBSCRIPTIONS=$(aws sns list-subscriptions-by-topic --output text --topic-arn $RESOURCE_ARN --query 'Subscriptions[*].{SubscriptionArn:SubscriptionArn}|[*].SubscriptionArn')
#    for subscriptionArn in $TOPIC_SUBSCRIPTIONS
#    do
#      aws sns unsubscribe --subscription-arn $subscriptionArn
#    done
#    aws sns delete-topic --topic-arn $RESOURCE_ARN
#    ;;
#
#  "AWS::SQS::Queue")
#    QUEUE_URL=$(aws sqs get-queue-url --queue-name $ARN_SUFFIX | jq -r -c '.QueueUrl')
#    aws sqs delete-queue --queue-url $QUEUE_URL
#    ;;
#
#  "AWS::ApiGateway::RestApi")
#    API_ID=$(echo $ARN_SUFFIX | sed -E 's/^.*\/([^\/]+)$/\1/')
#    aws apigateway delete-rest-api --rest-api-id $API_ID
#    ;;
#
#  "AWS::Cognito::UserPool")
#    USER_POOL_ID=$(echo $ARN_SUFFIX | sed -E 's/^.*\/([^\/]+)$/\1/')
#    aws cognito-idp delete-user-pool-domain --user-pool-id $USER_POOL_ID --domain dce-$ACCOUNT-$NAMESPACE
#    aws cognito-idp delete-user-pool --user-pool-id $USER_POOL_ID
#    ;;
#
#  "AWS::Cognito::IdentityPool")
#    IDENTITY_POOL_ID=$(echo $RESOURCE_ARN | sed -E 's/^.*\/([^\/]+)$/\1/')
#    aws cognito-identity delete-identity-pool --identity-pool-id $IDENTITY_POOL_ID
#    ;;
#
#  "AWS::Events::Rule")
#    RULE_NAME=$(echo $ARN_SUFFIX | sed -E 's/^.*\/([^\/]+)$/\1/')
#    TARGET_ID=$(aws events list-targets-by-rule --rule $RULE_NAME | jq -c -r '[.[] | .[]] | .[] | .Id')
#    aws events remove-targets --rule $RULE_NAME --ids $TARGET_ID
#    aws events delete-rule --name $RULE_NAME
#    ;;
#
#  "AWS::SES::ConfigurationSet")
#    aws ses delete-configuration-set --configuration-set-name $ARN_SUFFIX
#    ;;
#
#  "AWS::IAM::Policy")
#    aws iam delete-policy --policy-arn $RESOURCE_ARN
#    ;;
#
#  "AWS::SNS::Topic")
#    aws sns delete-topic --topic-arn $RESOURCE_ARN
#    ;;
#
#  "AWS::SSM::Parameter")
#    aws ssm delete-parameter --name $ARN_SUFFIX
#    ;;
#
#  "AWS::S3::Bucket")
#    echo "Skipping S3 bucket $RESOURCE_ARN."
#    ;;
#
#  *)
#    echo "Could not find delete command for $RESOURCE_ARN"
#    ;;
#esac
#
#}
#
################
## Validations #
################
#
#[ -z "$REGION" ] || [ -z "$NAMESPACE" ] && echo "Usage: ./delecte-dce.sh [region] [namespace] \
#
#  region: AWS region, e.g. us-east-1
#  namespace: The namespace appended to all of your DCE infrastructure. e.g. you should see a lambda named \"update_lease_status-{namespace}\"
#" && exit 1
#[ ! ${#NAMESPACE} -ge 4 ] && echo "Namespace must be longer than 3 characters" && exit 1
#[ $(command -v jq> /dev/null 2>&1; echo $?) -ne 0 ] && echo "Please install jq (https://stedolan.github.io/jq/)" && exit 1
#
#################################
## Get users consent to proceed #
#################################
#
#echo "\
#
#Warning: This script will delete infrastructure in whatever AWS account your AWS CLI is pointed at.
#
#DO NOT USE AGAINST A PRODUCTION ENVIRONMENT.
#
#Would you like to:
#- Delete resources tagged with AppName=DCE
#- Delete resources with names/arns containing the substring "$NAMESPACE"
#
#Type 'yes' to continue.
#"
#
#read RESPONSE
#if [ "$RESPONSE" != "yes" ]
#then
#    echo "User did type not 'yes', exiting..."
#    exit 1
#fi
#
################################################################
## Use resource groups to delete everything tagged AppName=DCE #
################################################################
#
#echo "Searching for resources by tag...
#"
#aws resource-groups create-group \
#    --name $RESOURCE_GROUP_NAME \
#    --resource-query '{"Type":"TAG_FILTERS_1_0", "Query":"{\"ResourceTypeFilters\":[\"AWS::AllSupported\"],\"TagFilters\":[{\"Key\":\"AppName\", \"Values\":[\"DCE\"]}]}"}'> /dev/null 2>&1
#
#RESOURCE_GROUP_RESOURCE_LIST=$(echo $(aws resource-groups list-group-resources --group-name $RESOURCE_GROUP_NAME) | jq -c '[.[] | .[]] | .[]')
#for item in $RESOURCE_GROUP_RESOURCE_LIST
#do
#  RESOURCE_TYPE=$(echo $item | jq -r -c '.ResourceType')
#  RESOURCE_ARN=$(echo $item | jq -r -c '.ResourceArn')
#  echo "- $RESOURCE_TYPE | $RESOURCE_ARN"
#done
#
## Delete resources that could be discovered via resource group
#for item in $RESOURCE_GROUP_RESOURCE_LIST
#do
#  RESOURCE_TYPE=$(echo $item | jq -r -c '.ResourceType')
#  RESOURCE_ARN=$(echo $item | jq -r -c '.ResourceArn')
#  delete_resource ${RESOURCE_TYPE} ${RESOURCE_ARN}
#done
#
#aws resource-groups delete-group --group-name $RESOURCE_GROUP_NAME> /dev/null 2>&1
#
###########################################
## Delete resources not in resource group #
###########################################
#
## API Gateway
#API_IDS=$(aws apigateway get-rest-apis --output text --query 'items[*].{id:id,name:name}|[?contains(name,`'$NAMESPACE'`)]|[*].id')
#for apiId in $API_IDS
#do
#  delete_resource AWS::ApiGateway::RestApi arn:aws:apigateway:$REGION::/restapis/$apiId
#done
#
## Cognito User Pool & Identity Pool
#DCE_USER_POOL_IDS=$(aws cognito-idp list-user-pools --max-results 60 --output text --query 'UserPools[*].{Name:Name,Id:Id}|[?contains(Name,`'$NAMESPACE'`)]|[*].Id')
#for userPoolId in $DCE_USER_POOL_IDS
#do
#  delete_resource AWS::Cognito::UserPool arn:aws:cognito-idp:$REGION:$ACCOUNT:userpool/$userPoolId
#done
#
#DCE_IDENTITY_IDS=$(aws cognito-identity list-identity-pools --max-results 60 --output text --query 'IdentityPools[*].{IdentityPoolName:IdentityPoolName,IdentityPoolId:IdentityPoolId}|[?contains(IdentityPoolName,`'$NAMESPACE'`)]|[*].IdentityPoolId')
#for identityPoolId in $DCE_IDENTITY_IDS
#do
#  delete_resource AWS::Cognito::IdentityPool arn:aws:cognito-identity:$REGION:$ACCOUNT:identitypool/$identityPoolId
#done
#
## Cloudwatch event rules, targets, and metric alarms
#DCE_EVENT_RULES=$(aws events list-rules --output text --query 'Rules[*].{Name:Name,Arn:Arn}|[?contains(Name,`'$NAMESPACE'`)]|[*].Arn')
#for ruleArn in $DCE_EVENT_RULES
#do
#  delete_resource AWS::Events::Rule $ruleArn
#done
#
## SES Configuration Sets
#CONFIGURATION_SETS=$(aws ses list-configuration-sets --output text --query 'ConfigurationSets[*].{Name:Name}|[?contains(Name,`'$NAMESPACE'`)]')
#for configSetName in $CONFIGURATION_SETS
#do
#  delete_resource AWS::SES::ConfigurationSet $configSetName
#done
#
## Not using delete_resource since cloudwatch alarms has bulk delete api
#CLOUDWATCH_ALARMS=$(aws cloudwatch describe-alarms --output text --query 'MetricAlarms[*].{AlarmName:AlarmName}|[?contains(AlarmName,`'$NAMESPACE'`)]')
#aws cloudwatch delete-alarms --alarm-names $CLOUDWATCH_ALARMS
#
## IAM policies, roles, and role attachments (Not using delete_resource since IAM delete APIs require multiple flags)
#DCE_IAM_ROLE_LIST=$(aws iam list-roles --no-paginate --output text --query 'Roles[*].{RoleName:RoleName}|[?contains(RoleName,`'$NAMESPACE'`)]')
#for roleName in $DCE_IAM_ROLE_LIST
#do
#  echo "Deleting policies for role: $roleName"
#  MANAGED_POLICY_LIST=$(aws iam list-attached-role-policies --role-name $roleName --output text --query 'AttachedPolicies[*].PolicyArn')
#  for policyArn in $MANAGED_POLICY_LIST
#  do
#    aws iam detach-role-policy --role-name $roleName --policy-arn $policyArn
#  done
#  INLINE_POLICY_LIST=$(aws iam list-role-policies --role-name $roleName --output text --query 'PolicyNames[*]')
#  for policyName in $INLINE_POLICY_LIST
#  do
#    aws iam delete-role-policy --role-name $roleName --policy-name $policyName
#  done
#  echo "Deleting role: $roleName"
#  aws iam delete-role --role-name $roleName
#done
#REMAINING_IAM_POLICIES=$(aws iam list-policies --output text --query 'Policies[*].{PolicyName:PolicyName,Arn:Arn}|[?contains(PolicyName,`'$NAMESPACE'`)]|[*].Arn')
#for policyArn in $REMAINING_IAM_POLICIES
#do
#  delete_resource AWS::IAM::Policy $policyArn
#done
#
## SNS Topics
#DCE_SNS_TOPICS=$(aws sns list-topics --output text --query 'Topics[*].{TopicArn:TopicArn}|[?contains(TopicArn,`'$NAMESPACE'`)]')
#for topicArn in $DCE_SNS_TOPICS
#do
#  delete_resource AWS::SNS::Topic $topicArn
#done
#
## SSM Parameters
#PARAMETER_NAME_LIST=$(aws ssm get-parameters-by-path --path "/"$NAMESPACE"/auth/" --query "Parameters[*].Name" --output text)
#for parameterName in $PARAMETER_NAME_LIST
#do
#  delete_resource AWS::SSM::Parameter $parameterName
#done
#
## Lambda Event Source Mappings
#DCE_LAMBDA_EVENT_SOURCEMAPPINGS=$(aws lambda list-event-source-mappings --output text --query 'EventSourceMappings[*].{UUID:UUID,FunctionArn:FunctionArn}|[?contains(FunctionArn,`'$NAMESPACE'`)]|[*].UUID')
#for mappingUuid in $DCE_LAMBDA_EVENT_SOURCEMAPPINGS
#do
#  delete_resource AWS::Lambda::EventSourceMapping $mappingUuid
#done
#
#echo " \
#
#FINISHED
#
#Please delete the following resources in the AWS console
#  - S3 bucket in the AWS console: $ACCOUNT-dce-artifacts-$NAMESPACE
#  - Any unwanted email addresses in Simple Email Service > Identity Management > Email Addresses
#"