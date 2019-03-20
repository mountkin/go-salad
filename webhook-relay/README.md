# GitHub webhook relay
This tool provides a GitHub webhook event sink that pushes the received messages to an AWS SQS queue.
A cooperating client side tool continuously reads messages from the queue and send it to
Jenkins webhook endpoint.

# Why need this?
If you want to trigger Jenkins jobs with GitHub webhook and keep the Jenkins server secure at the same time,
this tool is right for you.

As we know, exposing Jenkins to the public internet is extremely dangerous.
Many CVEs corresponding to Jenkins server had been reported recently.
[Some of the vulnerabilities](https://jenkins.io/security/advisory/2019-01-08/) are quite easy to be used by the attacker,
for example: https://github.com/adamyordan/cve-2019-1003000-jenkins-rce-poc

With this tool, you can deploy the Jenkins server in you private network behind the firewall
and eliminate most of the known or unknown vulnerabilities.

# Usage
1. Create an AWS account.
2. Install [serverless](https://serverless.com) CLI tool and create credentials following
    [this instruction](https://serverless.com/framework/docs/providers/aws/guide/quick-start/).
3. Edit `serverless.yml`, change the `__GITHUB_WEBHOOK_SECRET__` to a secure random string.
4. Run `make deploy` to deploy the lambda function to AWS. After the deploy, a Docker image named `mountkin/webhook-relay` will be created.
5. Configure the GitHub webhook URL according the output of step 4.
   Set the `Secret` field of the GitHub webhook to the random string that was generated in step 3.
6. Set the `Shared secret` field in the Jenkins system configuration page to the random string
   that was generated in step 3.
7. Run the following command to start the local webhook relay:
    ```bash
    docker run -d --restart=always --name=webhook-relay \
        -e AWS_REGION=ap-northeast-1 \
        -e QUEUE_URL=https://sqs.ap-northeast-1.amazonaws.com/xxxx/github-webhook \
        -e JENKINS_GITHUB_HOOK_URL=http://your-jenkins-server/ghprbhook/ \
        -e AWS_ACCESS_KEY_ID=your-aws-access-key \
        -e AWS_SECRET_ACCESS_KEY=your-aws-secret-key \
        mountkin/webhook-relay
    ```

    Note: be sure to change the environment variables accordingly.
    
    The `QUEUE_URL` can be found on the AWS SQS console.
    
    You can create a separate IAM role and grant only `sqs:ReceiveMessage` permission to the SQS queue
    and create an API key to use in the `docker run` command.

8. Create a GitHub pull request and see if the Jenkins job starts within 5 seconds.
