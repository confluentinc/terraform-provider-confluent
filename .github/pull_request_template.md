Release Notes
---------
<!-- If this PR introduces any user-facing changes, document them below as a summary. Delete unused section titles and placeholders. Match the style of previous release notes: https://github.com/confluentinc/terraform-provider-confluent/releases -->

New Features
- [Briefly describe new features introduced in this PR].

Bug Fixes
- [Briefly describe any bugs fixed in this PR].

Examples
- [Briefly describe any Terraform configuration example updates in this PR].

Checklist
---------
<!-- 
Check each item in the checklist to ensure high-quality Terraform development practices are followed. PR approval won't be granted until the checklist is carefully reviewed.
For instructions, please refer to this Confluence page: https://confluentinc.atlassian.net/wiki/spaces/AEGI/pages/3938058831/
-->
- [ ] I can successfully build and use a custom Terraform provider binary for Confluent.
- [ ] I have verified my PR with real Confluent Cloud resources in a pre-prod/production environment, or both.
- [ ] I have attached manual Terraform verification results or screenshots in the `Test & Review` section below.
- [ ] I have included appropriate Terraform acceptance or unit tests for any new resource, data source, or functionality.
- [ ] I confirm that this PR introduces no breaking changes or backward compatibility issues.
- [ ] I have updated the corresponding documentation and include relevant examples for this PR.
- [ ] I have indicated the potential customer(s) impact if something goes wrong in the `Blast Radius` section below.
- [ ] I have put checkmark below about the feature associated with this PR is enabled in:
  - [ ] Confluent Cloud prod
  - [ ] Confluent Cloud stag
  - [ ] Confluent Cloud devel
  - [ ] Check this box if the feature flag is enabled for certain organization only

What
----
<!--
Briefly describe **what** you have changed and **why** these changes are necessary.
Optionally include: 
- The problem being solved or the feature being added. 
- The implementation strategy or approach taken. 
- Key technical details, design decisions, or any additional context reviewers should be aware of.
-->

Blast Radius
----
<!--
The Blast Radius section should include information on what will be the customer(s) impact if something goes wrong or unexpectedly, 
adding this section will trigger the PR author to think about the impact from product perspective, examples can be:
- Confluent Cloud customers who are using `confluent_kafka_cluster` resource/data-source will be blocked.
- Confluent Cloud customers who are using `confluent_schema` resource for schema validation will be blocked.
- All customers who are using `terraform import` function for resources will be impacted.
-->

References
----------
<!-- Include links to relevant resources for this PR, such as: 
- Related GitHub issues 
- Tickets (JIRA, etc.) 
- Internal documentation or design specs 
- Other related PRs 
Copy and paste the links below for easy reference.
-->

Test & Review
-------------
<!-- Has this PR been tested? If so, explain **how** it was tested. Include: 
- Steps taken to verify the changes. 
- Links to manual verification documents, logs, or screenshots to save reviewers' time. 
- Any additional notes on testing (e.g., environments used, edge cases tested). 
- Screenshot showing successful resource creation.
Example: - [Manual Verification Document](https://docs.google.com/document/d/1dutVZmbEwJBBqMzx57uCXqllV1SEr2vxnjUrtTPCwBk/edit?tab=t.0#heading=h.6zajc95mev5j)
-->
