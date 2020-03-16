# iampolgen Tool

This tool is for generating both an example IAM policy and a list, in
[Markdown](https://en.wikipedia.org/wiki/Markdown) of the human-readable names
of the AWS services that are supported and not supported by [AWS
Nuke](https://github.com/rebuy-de/aws-nuke).

## AWS Nuke support

DCE currently uses AWS Nuke to clean up the resources in a leased account when
the lease is terminated and before it is returned to the account pool. As a
result, DCE requires delete support for resources in order to allow those
resources to be created to avoid the resources being left behind when the
account lease terminates.

AWS Nuke has a command `aws-nuke resource-types` that lists supported resources,
but the names of these resource types are free-form text that is up to the
developer to name--there is no mechanism that enforces the naming to make sure
it matches the name of an AWS service or operation.  For example:

```golang
func init() {
	register("RedshiftCluster", ListRedshiftClusters)
}
```

## iampolgen approach

The approach of the `iampolgen` tool is to scan the AWS Nuke `resources` folder
for handling resources. It looks through each file for a call to `DeleteX` or
`TerminateX`, and then looks at the services reference at the top of the file to
build a list of likely candidates for supported delete operations. Using this
list, it then looks through the `policy.js` file from AWS. It compares its list
of likely candidates with operations and service prefixes in that file to weed
out false positives, and from there generates a list of supported delete
operations with a pretty high degree of confidence.

For example, in that file to remove [Amazon
Redshift](https://aws.amazon.com/redshift/) clusters, `redshift-clusters.go`
(link to file in Github
[here](https://github.com/rebuy-de/aws-nuke/blob/master/resources/redshift-clusters.go))
see the following:

The **referenced import**:

```golang
import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/redshift"
)
```

The **delete call**:

```golang
func (f *RedshiftCluster) Remove() error {

	_, err := f.svc.DeleteCluster(&redshift.DeleteClusterInput{
		ClusterIdentifier:        f.clusterIdentifier,
		SkipFinalClusterSnapshot: aws.Bool(true),
	})

	return err
}
```

So the tool finds `redshift` and `DeleteCluster` to get `redshift:DeleteCluster`
and looks for them in the JSON:

```json
        "Amazon Redshift": {
            "StringPrefix": "redshift",
            "Actions": [
                "...snipped",
                "DeleteCluster",
                "...snipped",
```

The tool finds them and stores "Amazon Redshift" as a supported service and
"redshift:DeleteCluster" as a supported delete operation.

### Handling discrepencies

This approach is fairly accurate, but it's not perfect. AWS SDKs match up
*nearly* 100% between reference names and service prefixes. *Nearly*. To
accommodate the discrepencies (i.e., *elb* instead of *elasticloadbalancing*)
there is a map called `servicePrefixOverrides` that maps the service's reference
name to the IAM prefix.

### Updating the sources

There are two sources that should be updated in order to keep the generated 
markdown list and sample policy current: the [AWS Nuke source](https://github.com/rebuy-de/aws-nuke)
and the [policy file](https://awspolicygen.s3.amazonaws.com/js/policies.js)

Both the AWS Nuke source location and the path to the policies file are arguments
to the tool. So, in order to update the tool, run the following commands:

```bash
$ git clone https://github.com/rebuy-de/aws-nuke.git /path/to/local/aws-nuke
$ curl https://awspolicygen.s3.amazonaws.com/js/policies.js -o /path/to/policies.js
$ cat /path/to/policies.js | sed 's/app.PolicyEditorConfig=//' > /path/to/policy.json
# to generate the sample IAM policy (redirect STDOUT to a file if you wish)
$ iampolgen -nuke-source-dir=/path/to/local/aws-nuke -policies-js-file=/path/to/policy.json
# to generate the markdown list (redirect STDOUT to a file if you wish)
$ iampolgen -nuke-source-dir=/path/to/local/aws-nuke -policies-js-file=/path/to/policy.json -generate-markdown
```
