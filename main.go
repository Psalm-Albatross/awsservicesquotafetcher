package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
)

// ✅ Updated list of valid AWS services
// var validServiceCodes = map[string]string{
// 	"ec2":            "Amazon Elastic Compute Cloud (EC2)",
// 	"vpc":            "Amazon Virtual Private Cloud (VPC)",
// 	"route53":        "Amazon Route 53",
// 	"elb":            "Elastic Load Balancing (ELB)",
// 	"autoscaling":    "Auto Scaling",
// 	"inspector":      "Amazon Inspector",
// 	"apigateway":     "Amazon API Gateway",
// 	"dynamodb":       "Amazon DynamoDB",
// 	"ebs":            "Amazon Elastic Block Store (EBS)",
// 	"efs":            "Amazon Elastic File System (EFS)",
// 	"ecr":            "Amazon Elastic Container Registry (ECR)",
// 	"eks":            "Amazon Elastic Kubernetes Service (EKS)",
// 	"ses":            "Amazon Simple Email Service (SES)",
// 	"sns":            "Amazon Simple Notification Service (SNS)",
// 	"acm":            "AWS Certificate Manager (ACM)",
// 	"secretsmanager": "AWS Secrets Manager",
// 	"elasticip":      "Elastic IP",
// 	"backup":         "AWS Backup",
// 	"sqs":            "Amazon Simple Queue Service (SQS)",
// 	"kms":            "AWS Key Management Service (KMS)",
// 	"iam":            "AWS Identity and Access Management (IAM)",
// }

var validServiceCodes = map[string]string{
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
	// ✅ Added 20 More AWS Services Below
	"lambda":        "AWS Lambda",
	"rds":           "Amazon Relational Database Service (RDS)",
	"redshift":      "Amazon Redshift",
	"cloudfront":    "Amazon CloudFront",
	"cloudwatch":    "Amazon CloudWatch",
	"opensearch":    "Amazon OpenSearch Service",
	"s3":            "Amazon Simple Storage Service (S3)",
	"glacier":       "Amazon S3 Glacier",
	"sagemaker":     "Amazon SageMaker",
	"elasticache":   "Amazon ElastiCache",
	"codebuild":     "AWS CodeBuild",
	"codepipeline":  "AWS CodePipeline",
	"codedeploy":    "AWS CodeDeploy",
	"glue":          "AWS Glue",
	"athena":        "Amazon Athena",
	"stepfunctions": "AWS Step Functions",
	"msk":           "Amazon Managed Streaming for Apache Kafka (MSK)",
	"appmesh":       "AWS App Mesh",
	"timestream":    "Amazon Timestream",
	"fsx":           "Amazon FSx",
}

// ✅ Lists all valid AWS services
func listValidServices() {
	fmt.Println("✅ Valid AWS Services:")
	for code, name := range validServiceCodes {
		fmt.Printf("  %s - %s\n", code, name)
	}
	os.Exit(0)
}

// ✅ Fetch available quotas for a service
func listQuotasForService(serviceCode string, region string, profile string) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		log.Fatalf("AWS config error: %v", err)
	}

	svc := servicequotas.NewFromConfig(cfg)

	input := &servicequotas.ListServiceQuotasInput{
		ServiceCode: &serviceCode,
	}

	result, err := svc.ListServiceQuotas(context.TODO(), input)
	if err != nil {
		log.Fatalf("Error fetching quotas for %s: %v", serviceCode, err)
	}

	fmt.Printf("✅ Available Quotas for %s in region %s:\n", serviceCode, region)
	for _, quota := range result.Quotas {
		fmt.Printf("  - %s (Quota Code: %s)\n", *quota.QuotaName, *quota.QuotaArn)
	}
	os.Exit(0)
}

// ✅ Fetch AWS service quotas
func fetchQuotas(services []string, regions []string, profile string, outputFile string) {
	for _, region := range regions {
		log.Printf("Fetching quotas for services: %v in region: %s", services, region)

		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(region),
			config.WithSharedConfigProfile(profile),
		)
		if err != nil {
			log.Fatalf("AWS config error: %v", err)
		}

		svc := servicequotas.NewFromConfig(cfg)
		var records [][]string
		headers := []string{"Service Name", "Quota Name", "Region", "Current Quotas", "Utilized Quotas", "Used (%)"}
		records = append(records, headers)

		for _, serviceCode := range services {
			if _, valid := validServiceCodes[serviceCode]; !valid {
				log.Printf("Skipping invalid service: %s", serviceCode)
				fmt.Printf("❌ Invalid AWS Service Code: %s\n", serviceCode)
				continue
			}

			input := &servicequotas.ListServiceQuotasInput{
				ServiceCode: &serviceCode,
			}

			result, err := svc.ListServiceQuotas(context.TODO(), input)
			if err != nil {
				log.Printf("Error fetching quotas for %s: %v", serviceCode, err)
				fmt.Printf("❌ Error fetching quotas for %s: %v\n", serviceCode, err)
				continue
			}

			for _, quota := range result.Quotas {
				currentQuota := 0.0
				if quota.Value != nil {
					currentQuota = *quota.Value
				}

				row := []string{
					serviceCode,
					*quota.QuotaName,
					region, // ✅ Added Region Column
					strconv.FormatFloat(currentQuota, 'f', 2, 64),
					"0.00",  // Placeholder for used quota
					"0.00%", // Placeholder for percentage used
				}
				records = append(records, row)
			}
		}

		if outputFile != "" {
			saveToCSV(outputFile, records)
		} else {
			printTable(records)
		}
	}
}

// ✅ Save data to CSV
func saveToCSV(filename string, records [][]string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			log.Fatalf("Error writing to CSV: %v", err)
		}
	}

	log.Printf("Data saved to %s", filename)
}

// ✅ Print data in table format
func printTable(records [][]string) {
	for _, row := range records {
		fmt.Println(strings.Join(row, "\t"))
	}
}

func main() {
	// ✅ CLI flags
	servicesFlag := flag.String("services", "", "Comma-separated list of AWS services (e.g., ec2,vpc)")
	regionsFlag := flag.String("regions", "us-east-1", "Comma-separated AWS regions")
	profileFlag := flag.String("profile", "default", "AWS profile name")
	outputFlag := flag.String("output", "", "CSV output file (optional)")
	listServicesFlag := flag.Bool("list-services", false, "List all valid AWS service codes")
	listQuotasFlag := flag.String("list-quotas", "", "List quotas for a specific service (e.g., --list-quotas ec2)")

	flag.Parse()

	// ✅ List valid AWS services
	if *listServicesFlag {
		listValidServices()
	}

	// ✅ List quotas for a specific service
	if *listQuotasFlag != "" {
		service := *listQuotasFlag
		if _, valid := validServiceCodes[service]; !valid {
			fmt.Printf("❌ Invalid AWS Service Code: %s\n", service)
			listValidServices()
		} else {
			listQuotasForService(service, "us-east-1", *profileFlag)
		}
	}

	// ✅ Show help if no service is provided
	if *servicesFlag == "" {
		fmt.Println("Usage: go run main.go --services ec2,vpc --regions us-east-1 --profile default --output quotas.csv")
		fmt.Println("  --services     : Comma-separated list of AWS services to check quotas for (e.g., ec2,vpc)")
		fmt.Println("  --regions      : AWS region(s) (default: us-east-1)")
		fmt.Println("  --profile      : AWS profile to use for authentication (default: default)")
		fmt.Println("  --output       : Save the output as CSV (optional)")
		fmt.Println("  --list-services: List valid AWS services")
		fmt.Println("  --list-quotas  : List quotas for a service (e.g., --list-quotas ec2)")
		os.Exit(0)
	}

	services := strings.Split(*servicesFlag, ",")
	regions := strings.Split(*regionsFlag, ",")

	// ✅ Fetch quotas
	fetchQuotas(services, regions, *profileFlag, *outputFlag)
}
