# 🚀 AWS Services Quota Fetcher (`awsservicesquotafetcher`)

**`awsservicesquotafetcher`** is a CLI tool for monitoring and fetching AWS service quotas, including default and consumed quotas.

## 📥 Download & Install

### **1️⃣ Download the Binary**
Visit the [Releases](https://github.com/YOUR_GITHUB_REPO/releases) page to download the latest version for your operating system.

```
| OS           | Architecture | Binary Name |
|-------------|-------------|-------------------------------------|
| Linux       | amd64       | `awsservicesquotafetcher-0.1.0-linux-amd64` |
| Linux       | arm64       | `awsservicesquotafetcher-0.1.0-linux-arm64` |
| Windows     | amd64       | `awsservicesquotafetcher-0.1.0-windows-amd64.exe` |
| macOS       | amd64       | `aawsservicesquotafetcher-0.1.0-darwin-amd64` |
| macOS       | arm64 (M1/M2) | `awsservicesquotafetcher-0.1.0-darwin-arm64` |
```

> **Note:** Replace `VERSION` with the latest release version (e.g., `1.2.3`).

---

### **2️⃣ Grant Execution Permission (Linux/macOS)**
After downloading, navigate to the folder where the binary is stored and run:

```
chmod +x awsservicesquotafetcher-<VERSION>-<OS>-<ARCH>
```

Example for Linux:

```
chmod +x awsservicesquotafetcher-1.2.3-linux-amd64
```

### **3️⃣ Move the Binary to `/usr/local/bin/` (Optional)**
To use the tool system-wide:

```
sudo mv awsservicesquotafetcher-1.2.3-linux-amd64 /usr/local/bin/awsservicesquotafetcher
```

Now, you can run it from anywhere:

```
awsservicesquotafetcher --help
```

---

## 🚀 Usage

### **Check AWS Quotas for a Service**
```
awsservicesquotafetcher --service ec2
```

### **Check Quotas for Multiple Services**
```
awsservicesquotafetcher --services ec2,s3,rds
```

### **Check Quotas in a Specific AWS Region**
```
awsservicesquotafetcher --service s3 --region us-west-2
```

### **Use a Specific AWS Profile**
```
awsservicesquotafetcher --service vpc --profile my-aws-profile
```

### **Export Output to CSV**
```
awsservicesquotafetcher --services ec2,rds --output quotas.csv
```

### **Display Help**
```
awsservicesquotafetcher --help
```

---

## 🛠️ Troubleshooting

### **1. Permission Denied**
If you see a permission error:

```
chmod +x awsservicesquotafetcher-<VERSION>-<OS>-<ARCH>
```

### **2. "Command Not Found" Error**
Make sure the binary is in your `$PATH`. You can move it to `/usr/local/bin/`:

```
sudo mv awsservicesquotafetcher /usr/local/bin/
```

### **3. AWS Credentials Not Configured**
Ensure you have configured AWS credentials using:

```
aws configure
```

---

## 📌 License
This project is licensed under the **MIT License**.