package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/appmesh"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/fsx"
	"github.com/aws/aws-sdk-go-v2/service/glacier"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/inspector"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/timestreamwrite"
)

const (
	version = "1.0.0"
)

// QuotaInfo stores service quota details
type QuotaInfo struct {
	ServiceName  string
	QuotaName    string
	Region       string
	Allocated    float64
	Used         float64
	UtilizedPerc float64
}

// FetchServiceQuotas retrieves quota info for a given AWS service
func FetchServiceQuotas(ctx context.Context, cfg aws.Config, serviceCode string, region string) ([]QuotaInfo, error) {
	sqClient := servicequotas.NewFromConfig(cfg)

	// Fetch allocated quotas using Service Quotas API
	sqOutput, err := sqClient.ListServiceQuotas(ctx, &servicequotas.ListServiceQuotasInput{
		ServiceCode: aws.String(serviceCode),
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching quotas for %s: %v", serviceCode, err)
	}

	var quotas []QuotaInfo
	for _, quota := range sqOutput.Quotas {
		allocated := *quota.Value
		used := fetchUsedQuota(ctx, cfg, serviceCode, quota.QuotaName)

		utilized := 0.0
		if allocated > 0 {
			utilized = (used / allocated) * 100
		}

		quotas = append(quotas, QuotaInfo{
			ServiceName:  serviceCode,
			QuotaName:    *quota.QuotaName,
			Region:       region,
			Allocated:    allocated,
			Used:         used,
			UtilizedPerc: utilized,
		})
	}

	return quotas, nil
}

// Fetch actual usage based on service
func fetchUsedQuota(ctx context.Context, cfg aws.Config, serviceCode string, quotaName *string) float64 {
	switch serviceCode {
	case "rds":
		if *quotaName == "Parameter groups" {
			rdsClient := rds.NewFromConfig(cfg)
			output, err := rdsClient.DescribeDBParameterGroups(ctx, &rds.DescribeDBParameterGroupsInput{})
			if err == nil {
				return float64(len(output.DBParameterGroups))
			}
		}
	case "ec2":
		if *quotaName == "Running On-Demand instances" {
			ec2Client := ec2.NewFromConfig(cfg)
			output, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
			if err == nil {
				count := 0
				for _, res := range output.Reservations {
					count += len(res.Instances)
				}
				return float64(count)
			}
		}
	case "s3":
		if *quotaName == "Total Buckets" {
			s3Client := s3.NewFromConfig(cfg)
			output, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
			if err == nil {
				return float64(len(output.Buckets))
			}
		}
	case "vpc":
		if *quotaName == "VPCs per Region" {
			vpcClient := ec2.NewFromConfig(cfg)
			output, err := vpcClient.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
			if err == nil {
				return float64(len(output.Vpcs))
			}
		}
	case "route53":
		if *quotaName == "Hosted Zones" {
			route53Client := route53.NewFromConfig(cfg)
			output, err := route53Client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
			if err == nil {
				return float64(len(output.HostedZones))
			}
		}
	case "elasticloadbalancing":
		if *quotaName == "Load Balancers" {
			elbClient := elasticloadbalancing.NewFromConfig(cfg)
			output, err := elbClient.DescribeLoadBalancers(ctx, &elasticloadbalancing.DescribeLoadBalancersInput{})
			if err == nil {
				return float64(len(output.LoadBalancerDescriptions))
			}
		}
	case "autoscaling":
		if *quotaName == "Auto Scaling Groups" {
			asgClient := autoscaling.NewFromConfig(cfg)
			output, err := asgClient.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{})
			if err == nil {
				return float64(len(output.AutoScalingGroups))
			}
		}
	case "inspector":
		if *quotaName == "Assessment Templates" {
			inspectorClient := inspector.NewFromConfig(cfg)
			output, err := inspectorClient.ListAssessmentTemplates(ctx, &inspector.ListAssessmentTemplatesInput{})
			if err == nil {
				return float64(len(output.AssessmentTemplateArns))
			}
		}
	case "apigateway":
		if *quotaName == "APIs" {
			apigatewayClient := apigateway.NewFromConfig(cfg)
			output, err := apigatewayClient.GetRestApis(ctx, &apigateway.GetRestApisInput{})
			if err == nil {
				return float64(len(output.Items))
			}
		}
	case "dynamodb":
		if *quotaName == "Tables" {
			dynamodbClient := dynamodb.NewFromConfig(cfg)
			output, err := dynamodbClient.ListTables(ctx, &dynamodb.ListTablesInput{})
			if err == nil {
				return float64(len(output.TableNames))
			}
		}
	case "ebs":
		if *quotaName == "Volumes" {
			ebsClient := ec2.NewFromConfig(cfg)
			output, err := ebsClient.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{})
			if err == nil {
				return float64(len(output.Volumes))
			}
		}
	case "efs":
		if *quotaName == "File Systems" {
			efsClient := efs.NewFromConfig(cfg)
			output, err := efsClient.DescribeFileSystems(ctx, &efs.DescribeFileSystemsInput{})
			if err == nil {
				return float64(len(output.FileSystems))
			}
		}
	case "ecr":
		if *quotaName == "Repositories" {
			ecrClient := ecr.NewFromConfig(cfg)
			output, err := ecrClient.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{})
			if err == nil {
				return float64(len(output.Repositories))
			}
		}
	case "eks":
		if *quotaName == "Clusters" {
			eksClient := eks.NewFromConfig(cfg)
			output, err := eksClient.ListClusters(ctx, &eks.ListClustersInput{})
			if err == nil {
				return float64(len(output.Clusters))
			}
		}
	case "ses":
		if *quotaName == "Verified Email Addresses" {
			sesClient := ses.NewFromConfig(cfg)
			output, err := sesClient.ListIdentities(ctx, &ses.ListIdentitiesInput{
				IdentityType: "EmailAddress",
			})
			if err == nil {
				return float64(len(output.Identities))
			}
		}
	case "sns":
		if *quotaName == "Topics" {
			snsClient := sns.NewFromConfig(cfg)
			output, err := snsClient.ListTopics(ctx, &sns.ListTopicsInput{})
			if err == nil {
				return float64(len(output.Topics))
			}
		}
	case "acm":
		if *quotaName == "Certificates" {
			acmClient := acm.NewFromConfig(cfg)
			output, err := acmClient.ListCertificates(ctx, &acm.ListCertificatesInput{})
			if err == nil {
				return float64(len(output.CertificateSummaryList))
			}
		}
	case "secretsmanager":
		if *quotaName == "Secrets" {
			secretsManagerClient := secretsmanager.NewFromConfig(cfg)
			output, err := secretsManagerClient.ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
			if err == nil {
				return float64(len(output.SecretList))
			}
		}
	case "elasticip":
		if *quotaName == "Elastic IPs" {
			ec2Client := ec2.NewFromConfig(cfg)
			output, err := ec2Client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
			if err == nil {
				return float64(len(output.Addresses))
			}
		}
	case "backup":
		if *quotaName == "Backup Plans" {
			backupClient := backup.NewFromConfig(cfg)
			output, err := backupClient.ListBackupPlans(ctx, &backup.ListBackupPlansInput{})
			if err == nil {
				return float64(len(output.BackupPlansList))
			}
		}
	case "sqs":
		if *quotaName == "Queues" {
			sqsClient := sqs.NewFromConfig(cfg)
			output, err := sqsClient.ListQueues(ctx, &sqs.ListQueuesInput{})
			if err == nil {
				return float64(len(output.QueueUrls))
			}
		}
	case "kms":
		if *quotaName == "Keys" {
			kmsClient := kms.NewFromConfig(cfg)
			output, err := kmsClient.ListKeys(ctx, &kms.ListKeysInput{})
			if err == nil {
				return float64(len(output.Keys))
			}
		}
	case "iam":
		if *quotaName == "Users" {
			iamClient := iam.NewFromConfig(cfg)
			output, err := iamClient.ListUsers(ctx, &iam.ListUsersInput{})
			if err == nil {
				return float64(len(output.Users))
			}
		}
	case "lambda":
		if *quotaName == "Functions" {
			lambdaClient := lambda.NewFromConfig(cfg)
			output, err := lambdaClient.ListFunctions(ctx, &lambda.ListFunctionsInput{})
			if err == nil {
				return float64(len(output.Functions))
			}
		}
	case "redshift":
		if *quotaName == "Clusters" {
			redshiftClient := redshift.NewFromConfig(cfg)
			output, err := redshiftClient.DescribeClusters(ctx, &redshift.DescribeClustersInput{})
			if err == nil {
				return float64(len(output.Clusters))
			}
		}
	case "cloudfront":
		if *quotaName == "Distributions" {
			cloudfrontClient := cloudfront.NewFromConfig(cfg)
			output, err := cloudfrontClient.ListDistributions(ctx, &cloudfront.ListDistributionsInput{})
			if err == nil {
				return float64(len(output.DistributionList.Items))
			}
		}
	case "cloudwatch":
		if *quotaName == "Alarms" {
			cloudwatchClient := cloudwatch.NewFromConfig(cfg)
			output, err := cloudwatchClient.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{})
			if err == nil {
				return float64(len(output.MetricAlarms))
			}
		}
	case "opensearch":
		if *quotaName == "Domains" {
			opensearchClient := opensearch.NewFromConfig(cfg)
			output, err := opensearchClient.ListDomainNames(ctx, &opensearch.ListDomainNamesInput{})
			if err == nil {
				return float64(len(output.DomainNames))
			}
		}
	case "glacier":
		if *quotaName == "Vaults" {
			glacierClient := glacier.NewFromConfig(cfg)
			output, err := glacierClient.ListVaults(ctx, &glacier.ListVaultsInput{})
			if err == nil {
				return float64(len(output.VaultList))
			}
		}
	case "sagemaker":
		if *quotaName == "Notebook Instances" {
			sagemakerClient := sagemaker.NewFromConfig(cfg)
			output, err := sagemakerClient.ListNotebookInstances(ctx, &sagemaker.ListNotebookInstancesInput{})
			if err == nil {
				return float64(len(output.NotebookInstances))
			}
		}
	case "elasticache":
		if *quotaName == "Clusters" {
			elasticacheClient := elasticache.NewFromConfig(cfg)
			output, err := elasticacheClient.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{})
			if err == nil {
				return float64(len(output.CacheClusters))
			}
		}
	case "codebuild":
		if *quotaName == "Projects" {
			codebuildClient := codebuild.NewFromConfig(cfg)
			output, err := codebuildClient.ListProjects(ctx, &codebuild.ListProjectsInput{})
			if err == nil {
				return float64(len(output.Projects))
			}
		}
	case "codepipeline":
		if *quotaName == "Pipelines" {
			codepipelineClient := codepipeline.NewFromConfig(cfg)
			output, err := codepipelineClient.ListPipelines(ctx, &codepipeline.ListPipelinesInput{})
			if err == nil {
				return float64(len(output.Pipelines))
			}
		}
	case "codedeploy":
		if *quotaName == "Applications" {
			codedeployClient := codedeploy.NewFromConfig(cfg)
			output, err := codedeployClient.ListApplications(ctx, &codedeploy.ListApplicationsInput{})
			if err == nil {
				return float64(len(output.Applications))
			}
		}
	case "glue":
		if *quotaName == "Jobs" {
			glueClient := glue.NewFromConfig(cfg)
			output, err := glueClient.GetJobs(ctx, &glue.GetJobsInput{})
			if err == nil {
				return float64(len(output.Jobs))
			}
		}
	case "athena":
		if *quotaName == "Workgroups" {
			athenaClient := athena.NewFromConfig(cfg)
			output, err := athenaClient.ListWorkGroups(ctx, &athena.ListWorkGroupsInput{})
			if err == nil {
				return float64(len(output.WorkGroups))
			}
		}
	case "stepfunctions":
		if *quotaName == "State Machines" {
			stepfunctionsClient := sfn.NewFromConfig(cfg)
			output, err := stepfunctionsClient.ListStateMachines(ctx, &sfn.ListStateMachinesInput{})
			if err == nil {
				return float64(len(output.StateMachines))
			}
		}
	case "appmesh":
		if *quotaName == "Meshes" {
			appmeshClient := appmesh.NewFromConfig(cfg)
			output, err := appmeshClient.ListMeshes(ctx, &appmesh.ListMeshesInput{})
			if err == nil {
				return float64(len(output.Meshes))
			}
		}
	case "timestream":
		if *quotaName == "Databases" {
			timestreamClient := timestreamwrite.NewFromConfig(cfg)
			output, err := timestreamClient.ListDatabases(ctx, &timestreamwrite.ListDatabasesInput{})
			if err == nil {
				return float64(len(output.Databases))
			}
		}
	case "fsx":
		if *quotaName == "File Systems" {
			fsxClient := fsx.NewFromConfig(cfg)
			output, err := fsxClient.DescribeFileSystems(ctx, &fsx.DescribeFileSystemsInput{})
			if err == nil {
				return float64(len(output.FileSystems))
			}
		}
	}

	// Default case returns 0.0
	return 0.0
}

// Save results to CSV
func SaveToCSV(quotas []QuotaInfo, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Service Name", "Quota Name", "Region", "Allocated Quota", "Used Quota", "Utilized (%)"})

	for _, q := range quotas {
		writer.Write([]string{
			q.ServiceName,
			q.QuotaName,
			q.Region,
			strconv.FormatFloat(q.Allocated, 'f', 2, 64),
			strconv.FormatFloat(q.Used, 'f', 2, 64),
			strconv.FormatFloat(q.UtilizedPerc, 'f', 2, 64) + "%",
		})
	}

	log.Printf("✅ CSV file saved to %s", outputPath)
	return nil
}

// List valid AWS services
func listValidServices() {
	validServiceCodes := map[string]string{
		"ec2":            "Amazon Elastic Compute Cloud (EC2)",
		"vpc":            "Amazon Virtual Private Cloud (VPC)",
		"route53":        "Amazon Route 53",
		"elb":            "Elastic Load Balancing (ELB)",
		"autoscaling":    "Auto Scaling",
		"inspector":      "Amazon Inspector",
		"apigateway":     "Amazon API Gateway",
		"dynamodb":       "Amazon DynamoDB",
		"ebs":            "Amazon Elastic Block Store (EBS)",
		"efs":            "Amazon Elastic File System (EFS)",
		"ecr":            "Amazon Elastic Container Registry (ECR)",
		"eks":            "Amazon Elastic Kubernetes Service (EKS)",
		"ses":            "Amazon Simple Email Service (SES)",
		"sns":            "Amazon Simple Notification Service (SNS)",
		"acm":            "AWS Certificate Manager (ACM)",
		"secretsmanager": "AWS Secrets Manager",
		"elasticip":      "Elastic IP",
		"backup":         "AWS Backup",
		"sqs":            "Amazon Simple Queue Service (SQS)",
		"kms":            "AWS Key Management Service (KMS)",
		"iam":            "AWS Identity and Access Management (IAM)",
		"lambda":         "AWS Lambda",
		"rds":            "Amazon Relational Database Service (RDS)",
		"redshift":       "Amazon Redshift",
		"cloudfront":     "Amazon CloudFront",
		"cloudwatch":     "Amazon CloudWatch",
		"opensearch":     "Amazon OpenSearch Service",
		"s3":             "Amazon Simple Storage Service (S3)",
		"glacier":        "Amazon S3 Glacier",
		"sagemaker":      "Amazon SageMaker",
		"elasticache":    "Amazon ElastiCache",
		"codebuild":      "AWS CodeBuild",
		"codepipeline":   "AWS CodePipeline",
		"codedeploy":     "AWS CodeDeploy",
		"glue":           "AWS Glue",
		"athena":         "Amazon Athena",
		"stepfunctions":  "AWS Step Functions",
		"msk":            "Amazon Managed Streaming for Apache Kafka (MSK)",
		"appmesh":        "AWS App Mesh",
		"timestream":     "Amazon Timestream",
		"fsx":            "Amazon FSx",
	}

	fmt.Println("Valid AWS services:")
	for code, name := range validServiceCodes {
		fmt.Printf("  %s - %s\n", code, name)
	}
	os.Exit(0)
}

// List quotas for a specific service
func listQuotasForService(serviceCode string, region string, profile string) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		log.Fatalf("❌ AWS config error: %v", err)
	}

	svc := servicequotas.NewFromConfig(cfg)

	input := &servicequotas.ListServiceQuotasInput{
		ServiceCode: &serviceCode,
	}

	result, err := svc.ListServiceQuotas(context.TODO(), input)
	if err != nil {
		log.Fatalf("❌ Error fetching quotas for %s: %v", serviceCode, err)
	}

	fmt.Printf("Available Quotas for %s in region %s:\n", serviceCode, region)
	for _, quota := range result.Quotas {
		fmt.Printf("  - %s (Quota Code: %s)\n", *quota.QuotaName, *quota.QuotaArn)
	}
	os.Exit(0)
}

// Push data to Slack
func pushToSlack(url string, data interface{}, format string) error {
	var payload []byte
	var err error

	if format == "json" {
		payload, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("❌ Error marshaling data to JSON: %v", err)
		}
	} else {
		var buffer bytes.Buffer
		for _, q := range data.([]QuotaInfo) {
			buffer.WriteString(fmt.Sprintf("%s\t%s\t%s\t%.2f\t%.2f\t%.2f%%\n", q.ServiceName, q.QuotaName, q.Region, q.Allocated, q.Used, q.UtilizedPerc))
		}
		payload = buffer.Bytes()
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("❌ Error creating HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("❌ Error sending HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("❌ Received non-OK response from Slack: %s", resp.Status)
	}

	return nil
}

func pushDataToSlack(url string, token string, data []QuotaInfo) error {
	var buffer bytes.Buffer
	buffer.WriteString("Service Name\tQuota Name\tRegion\tAllocated Quota\tUsed Quota\tUtilized (%)\n")
	for _, q := range data {
		buffer.WriteString(fmt.Sprintf("%s\t%s\t%s\t%.2f\t%.2f\t%.2f%%\n", q.ServiceName, q.QuotaName, q.Region, q.Allocated, q.Used, q.UtilizedPerc))
	}

	payload := map[string]string{"text": buffer.String()}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("❌ Error marshaling payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("❌ Error creating HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("❌ Error sending HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("❌ Received non-OK response from Slack: %s", resp.Status)
	}

	return nil
}

func main() {
	servicesFlag := flag.String("services", "", "Comma-separated AWS services (e.g., rds,ec2)")
	regionsFlag := flag.String("regions", "us-east-1", "Comma-separated AWS regions")
	profileFlag := flag.String("profile", "", "AWS profile name (required)")
	outputFlag := flag.String("output", "", "CSV output file (optional)")
	versionFlag := flag.Bool("version", false, "Display CLI version")
	listServicesFlag := flag.Bool("list-services", false, "List valid AWS services")
	listQuotasFlag := flag.String("list-quotas", "", "List quotas for a service (e.g., --list-quotas ec2)")
	slackURLFlag := flag.String("url-to-push", "", "Slack URL to push data")
	formatFlag := flag.String("format", "table", "Format to push data (table or json)")
	pushDataToSlackFlag := flag.String("push-data-to-slack", "", "Slack URL to push data")
	slackTokenFlag := flag.String("slack-token", "", "Slack API token for authentication")
	logFileFlag := flag.String("log-file", "awsservicesquotafetcher.log", "Log file path")

	flag.Parse()

	// Initialize logging
	logFile, err := os.OpenFile(*logFileFlag, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("❌ Error opening log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("🚀 Starting awsservicesquotafetcher")

	if *versionFlag {
		fmt.Println("awsservicesquotafetcher")
		fmt.Println("version:", version)
		fmt.Println("Developed by ChatGPT, Instructed By Psalm Albatross")
		log.Println("ℹ️ Displayed version information")
		return
	}

	if *listServicesFlag {
		listValidServices()
	}

	if *listQuotasFlag != "" {
		if *profileFlag == "" {
			log.Fatal("❌ Error: --profile flag is required")
		}
		service := *listQuotasFlag
		listQuotasForService(service, strings.Split(*regionsFlag, ",")[0], *profileFlag)
	}

	// Show help if no service is provided
	if *servicesFlag == "" {
		fmt.Println("Usage: go run main.go --services ec2,vpc --regions us-east-1 --profile default --output quotas.csv")
		fmt.Println("  --services         : Comma-separated list of AWS services to check quotas for (e.g., ec2,vpc)")
		fmt.Println("  --regions          : AWS region(s) (default: us-east-1)")
		fmt.Println("  --profile          : AWS profile to use for authentication (required)")
		fmt.Println("  --output           : Save the output as CSV (optional)")
		fmt.Println("  --list-services    : List valid AWS services")
		fmt.Println("  --list-quotas      : List quotas for a service (e.g., --list-quotas ec2)")
		fmt.Println("  --url-to-push      : Slack URL to push data")
		fmt.Println("  --format           : Format to push data (table or json)")
		fmt.Println("  --push-data-to-slack: Slack URL to push data")
		fmt.Println("  --slack-token      : Slack API token for authentication (required when using --push-data-to-slack)")
		log.Println("ℹ️ Displayed usage information")
		os.Exit(0)
	}

	if *profileFlag == "" {
		log.Fatal("❌ Error: --profile flag is required")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(*profileFlag),
		config.WithRegion(strings.Split(*regionsFlag, ",")[0]),
	)
	if err != nil {
		log.Fatalf("❌ Error loading AWS config: %v", err)
	}

	var allQuotas []QuotaInfo
	services := strings.Split(*servicesFlag, ",")
	regions := strings.Split(*regionsFlag, ",")

	for _, service := range services {
		for _, region := range regions {
			cfg.Region = region
			log.Printf("🔍 Fetching quotas for %s in region: %s", service, region)

			quotas, err := FetchServiceQuotas(context.TODO(), cfg, service, region)
			if err != nil {
				log.Printf("❌ Error fetching quotas for %s: %v", service, err)
				continue
			}
			allQuotas = append(allQuotas, quotas...)
		}
	}

	fmt.Println("Service Name\tQuota Name\tRegion\tAllocated Quota\tUsed Quota\tUtilized (%)")
	for _, q := range allQuotas {
		fmt.Printf("%s\t%s\t%s\t%.2f\t%.2f\t%.2f%%\n", q.ServiceName, q.QuotaName, q.Region, q.Allocated, q.Used, q.UtilizedPerc)
	}

	if *outputFlag != "" {
		if err := SaveToCSV(allQuotas, *outputFlag); err != nil {
			log.Fatalf("❌ Error saving CSV: %v", err)
		}
		log.Printf("✅ Saved quotas to CSV file: %s", *outputFlag)
	}

	if *slackURLFlag != "" {
		if err := pushToSlack(*slackURLFlag, allQuotas, *formatFlag); err != nil {
			log.Fatalf("❌ Error pushing data to Slack: %v", err)
		}
		log.Println("✅ Pushed data to Slack")
	}

	if *pushDataToSlackFlag != "" {
		if *slackTokenFlag == "" {
			log.Fatal("❌ Error: --slack-token flag is required when using --push-data-to-slack")
		}
		if err := pushDataToSlack(*pushDataToSlackFlag, *slackTokenFlag, allQuotas); err != nil {
			log.Fatalf("❌ Error pushing data to Slack: %v", err)
		}
		log.Println("✅ Pushed data to Slack with token")
	}

	log.Println("🏁 Finished awsservicesquotafetcher")
}
