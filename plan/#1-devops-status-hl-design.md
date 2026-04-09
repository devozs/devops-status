# DevOps Status
Simple layout with the following pages:
- Home
    - Main Status: box message of "All Systems Operational" or "Disruption with some Devops services"
    - Services breakdown: colored bar for each configured service
- History - tabs of Incidents and Uptime
Web app for showing devops services status, simmilar to:
- GitHub Service Status https://www.githubstatus.com/
- https://status.claude.com/

## Frontend
using Nuxt 4

## Backend
TBD, not sure what to use as BE.
Also need to consider and take into account the section of Data Srouce.
we can either use node, Go or other, but its important to make sure the BE can easily integrate with Nuxt FE.
consider all of the requirments in this HL plan and suggest accordingly.

## Persistence
preferend some SQL (like postgres) else if there is a better suggestion to persist the data per this requirment document.
We need to keep history of Service Avaliability and QOS including relevant metadata.
Metadata like date, incident details, relevant service, trends of avalibitly and QOS, uptime percentage, past incidents.
consider all of the requirments in this HL plan and suggest accordingly.

## Data Source
This is important part, there are few ways for data extraction that need to be supported.
Admin to be able to create a Data Source, one of two types: Operational or QOS.
Important items to consider for data extraction:
- most of the sevices are deployed in kubernetes. we provition Gaudi and CPU resources over k8s via kubevirt
- the users can consume these k8s resources using a dedicated CLI tool (implemented in GO using cobra)
- heavy use of prometheus so need to be able to extract data from it
- many custom services and prometheus exporters are implemented in GO Lang.
these exporters prepare data to be used later by Prometheus / Grafana
- The CLI tool used to consume k8s resources also to be used for Service Avaliability and QOS (i.e. try to provision a resource to determine if service avalible + the CLI tool print out timestamps for the varoius steps so it can be used to determine the QOS). As such our status webapp backend need to be able to execute bash
- As the CLI tool written in GO our status webapp backend need to be able execute Go Lang methods (of the existing implementation)
- Our status webapp backend need to be able to execute curl (or simmilar), its basicaly a more spesific case of bash but using curl we can get Service Avaliability and QOS of the relevant services

## Services Status
Admin to be able to create a service and for each to select how to check if its Operational and how to check its QOS out of the (already defined) supported Data Source.
There are two main indications:
- Service Avaliability - a call to determine if sevice is Operational or not
- Quality of service - if the service is up the admin can set levels to indicate the health of the serivce. red, yellow and green. Admin can determine pass rate per level.

Addtition admin should be able to set Service Avaliability and Quality of service sample rate (duration between samples and how many samples before setting up/down or QOS)

## External Integrations - Subscription
- Teams: first phase will include status udate for Microsoft Teams
- Email: first phase will include status udate for email
- later we`ll add more clients to consume the status so the app need to consider this future enhancment

## Deployment
currently out of scope, we`ll handle it later.

## Visual References:
can be found at directory ./plan/examples