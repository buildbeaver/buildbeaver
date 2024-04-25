# Known Issues

## Destroy fails due to Lambda

See https://github.com/hashicorp/terraform-provider-aws/issues/1721#issuecomment-921634211

TL;DR: Deleting Lambda@Edge functions sucks and we should figure out if there's a better way to achieve pretty react routes with CloudFront (or move away from CloudFront)

Work around is to manually delete the function association from the CloudFront distribution, wait ~20min, then run the destroy.