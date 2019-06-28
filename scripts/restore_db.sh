#!/bin/bash
set -euo pipefail

# restore_db.sh
# This script will create a DynamoDB Table from a restore on an existing Backup.
# NOTE: If there's an existing DynamoDB Table with the same name, the Table will 
# need to be deleted before the backup can be started. 
# Arguments:
#   -t|--target-table-name - Name of the DynamoDB Table to create
#   -l|--list-backups - Flag to list backups available for the DynamoDB Table provided
#   -b|--backup-arn - ARN of the DynamoDB Table Backup to restore from
#   -f|--force-delete-table - Flag to indicate a deletion of the DynamoDB Table if it exists
# 
# Example:
#   # List available backups for table 
#   ./restore_db.sh --target-table-name my-table-name --list-backups
#   # Target table doesn't exist, restore
#   ./restore_db.sh --target-table-name my-table-name --backup-arn arn:aws:dynamodb:us-east-1:123456789012:table/my-table-name/backup/5678901234-987654
#   # Target table exists, force delete the table, restore
#   ./restore_db.sh --target-table-name my-table-name --backup-arn arn:aws:dynamodb:us-east-1:123456789012:table/my-table-name/backup/5678901234-987654 --force-delete-table
# 
# Required tools on execution host:
# Bourne Again SHell (documented for completeness)
# aws - AWS Command Line Interface - https://aws.amazon.com/cli/

# Setup Parameters
TARGET_TABLE_NAME=""
LIST_BACKUPS=""
BACKUP_ARN=""
FORCE_DELETE_TABLE=""
PARAMS=""
while (( "$#" )); do
  case "$1" in
    -t|--target-table-name)
      TARGET_TABLE_NAME=$2
      shift 2
      ;;
    -l|--list-backups)
      LIST_BACKUPS=true
      shift 1
      ;;
    -b|--backup-arn)
      BACKUP_ARN=$2
      shift 2
      ;;
    -f|--force-delete-table)
      FORCE_DELETE_TABLE=true
      shift 1
      ;;
    -*|--*=) # unsupported flags
      echo "Error: Unsupported flag $1" >&2
      exit 1
      ;;
    *) # preserve positional arguments
      PARAMS="$PARAMS $1"
      shift
      ;;
  esac
done

# Verify Parameters
if test -z "$TARGET_TABLE_NAME"; then
    echo "Please provide the argument '-t' or '--target-table-name' with the DynamoDB Table Name to restore to"
    exit 1
fi
if test ! -z "$LIST_BACKUPS"; then
    echo "Listing backups for $TARGET_TABLE_NAME"
    aws dynamodb list-backups --table-name $TARGET_TABLE_NAME
    exit 0
fi
if test -z "$BACKUP_ARN"; then
    echo "Please provide the arguments '-b|--backup-arn' to backup the DynamoDB Table from an existing Backup"
    echo "Listing backups for $TARGET_TABLE_NAME"
    aws dynamodb list-backups --table-name $TARGET_TABLE_NAME
    exit 1
fi

# Check if Backup Exists
echo "Checking Backup $BACKUP_ARN exists.."
if aws dynamodb describe-backup --backup-arn $BACKUP_ARN; then
    # Backup Exists
    echo "Backup $BACKUP_ARN exists"
else 
    # Backup Doesn't Exist, exit
    echo "Backup $BACKUP_ARN doesn't exist, exiting"
    exit 1
fi

# Check if Table Exists, Remove Table if so
echo "Checking if Table $TARGET_TABLE_NAME exists"
if aws dynamodb describe-table --table-name $TARGET_TABLE_NAME; then
    echo "Table $TARGET_TABLE_NAME exists"

    # Check if force flag is enable 
    if test -z "$FORCE_DELETE_TABLE"; then 
        # Force delete flag must be provided, exit
        echo "Cannot delete $TARGET_TABLE_NAME without 'f|--force-delete-table' flag, exiting"
        exit 1
    fi
    
    # Delete Table
    echo "Deleting Table $TARGET_TABLE_NAME..."
    aws dynamodb delete-table --table-name $TARGET_TABLE_NAME

    # Loop until dynamodb is deleted
    while aws dynamodb describe-table --table-name $TARGET_TABLE_NAME > /dev/null; do
        echo "Waiting for deletion to complete..."
        sleep 5
    done
    echo "Deleting Table $TARGET_TABLE_NAME complete"
else
    # Table doesn't exist, no deletion
    echo "Table $TARGET_TABLE_NAME doesn't exist"
fi

# Restore DynamoDB Table to the latest restorable time
echo "Start Restoration of Table $TARGET_TABLE_NAME with $BACKUP_ARN..."
aws dynamodb restore-table-from-backup \
    --target-table-name $TARGET_TABLE_NAME \
    --backup-arn $BACKUP_ARN
echo "Restoring Table $TARGET_TABLE_NAME with $BACKUP_ARN started"
echo "Check Table Status with command 'aws dynamodb describe-table --table-name $TARGET_TABLE_NAME'"