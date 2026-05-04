import * as cdk from 'aws-cdk-lib';
import { NodejsFunction } from 'aws-cdk-lib/aws-lambda-nodejs';
import { Construct } from 'constructs';

export class AppStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // Add Lambda functions here using NodejsFunction, e.g.:
    // new NodejsFunction(this, 'GetUser', {
    //   entry: 'packages/get-user/index.ts',
    //   handler: 'handler',
    // });
  }
}
