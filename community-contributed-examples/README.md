# Community Contributed Examples

Welcome to the community-contributed examples directory! This section contains Terraform configurations contributed by the community to help address specific use cases not covered in the official Terraform Provider for Confluent documentation.

## ⚠️ Important Disclaimer

**These examples are community-contributed and unofficial.** They are provided as-is and may:
- Become outdated as the Terraform Provider for Confluent evolves
- Not follow the latest best practices
- Require modifications to work with your specific environment
- Not be maintained by the Confluent team

**Use these examples at your own discretion and always test thoroughly in a NON-PRODUCTION environment first.**

## 📋 How to Use These Examples

1. **Review the example**: Each contribution includes details about the use case, testing status, and any known limitations
2. **Test in your environment**: Always validate examples in a development environment before using in production
3. **Adapt as needed**: Modify the configurations to match your specific requirements
4. **Check for updates**: Ensure you're using compatible versions of the Terraform Provider for Confluent

## 🤝 Contributing Your Examples

We welcome community contributions! If you have a working Terraform configuration that addresses a specific use case, please consider sharing it with the community.

### Contribution Process

1. **Clone this repository** and create a new branch for your contribution
2. **Create your example directory** under `community-contributed-examples/` with a descriptive name
3. **Include your Terraform files** (`.tf` files) and any supporting documentation
4. **Use the PR template** when submitting your pull request (see [contribution_pr_template.md](./contribution_pr_template.md))
5. **Submit your PR** for review by the API team


## 📁 Example Structure

Each contributed example should follow a structure similar to below:

```
community-contributed-examples/
└── your-example-name/
    ├── README.md             # Explanation of the use case
    ├── main.tf               # Main Terraform configuration
    ├── variables.tf          # Variable definitions (if applicable)
    ├── outputs.tf            # Output definitions (if applicable)
    ├── terraform.tfvars.example  # Example variable values (if applicable)
    └── versions.tf           # Provider version constraints (if applicable)
```

## 🏷️ Categories

Examples are organized by use case and functionality. Each example name should be descriptive of the category/scenario it addresses. Common categories include:

- **Authentication & Security**: OAuth, RBAC, ACLs, API keys
- **Networking**: Private Link, VPC peering, network configurations
- **Kafka Management**: Cluster setup, topic management, configurations
- **Schema Registry**: Schema management, compatibility settings
- **Connect**: Connector configurations, custom plugins
- **ksqlDB**: ksqlDB cluster and application setups
- **Flink**: Flink compute pools, statements, and applications
- **Tableflow**: Tableflow topics, catalog integrations 
- **Other** 

## 📞 Support

For questions about these community examples:

1. **Check the example's README**: Most examples include troubleshooting tips
2. **Review the PR discussion**: Check the original pull request for additional context
3. **Ask the community**: Use the #terraform channel in Confluent Slack
4. **Official documentation**: Refer to the [official Terraform Provider documentation](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs)

## 🔄 Maintenance

This directory is maintained on a quarterly basis. Examples may be:
- Updated for compatibility with newer provider versions
- Archived if they become obsolete
- Reorganized for better discoverability

## 📜 License

These community examples are shared under the same license as the main repository. By contributing, you agree to license your contribution under these terms.

---

**Happy Terraforming!** 🚀

*For official examples and documentation, please visit the main [examples](../examples/) directory and the [Terraform Provider for Confluent documentation](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs).*