# CloudHSM Operator

Given the CloudHSM cluster ID, CloudHSM Operator runs a reconciler loop to
create and update the ConfigMap object with the CloudHSM IP addresses currently
present inthe CloudHSM cluster. 

The example deploy uses IAM Roles for Service Accounts, EKS specific
functionality to authorize CloudHSM Operator against AWS API.

TODO:
- Provision CloudHSM devices based on the given size input parameter
- Filter list of HSM to contain only devices in active state, e.g., not
  currently being deleted
- Provide end-to-end example how to use CloudHSM Operator on AWS
