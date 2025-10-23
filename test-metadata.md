---
organization: P1174
project: Cloud-Migration
context: BACKEND
tags: test, demo
---

# Task Metadata Test File

This file tests all metadata syntax variants.

## Frontmatter Inheritance Test

These tasks should inherit organization=P1174, project=Cloud-Migration, context=BACKEND from frontmatter:

- [ ] Task with no inline metadata
- [ ] Task with priority ⏫ and date 📅 2025-12-01

## Inline Organization Test

- [ ] [DANORAMA] Company task with organization only
- [ ] [P1174] Client task with organization
- [ ] [INTERNAL] Internal work task

## Organization + Context Test

- [ ] [P1174:FINANCE] Financial report task
- [ ] [P1174:BACKEND] Backend development work
- [ ] [DANORAMA:HR] HR recruitment task

## Organization + Project + Context Test

- [ ] [P1174:Cloud:BACKEND] Full three-part syntax
- [ ] [P1174:Migration:DATABASE] Database migration task
- [ ] [DANORAMA:ERP:FINANCE] Company ERP finance module

## Key:Value Syntax Test

- [ ] Review code org:P1174 ctx:BACKEND
- [ ] Monthly report org:DANORAMA ctx:FINANCE
- [ ] Deploy service org:P1174 ctx:DEVOPS proj:Cloud
- [ ] Personal health task org:PERSONAL ctx:HEALTH

## Indented Metadata Test

- [ ] Task with indented metadata
  organization:: CLIENT-XYZ
  context:: RESEARCH
  project:: Data-Analysis
  due:: 2025-11-20
  priority:: high

- [ ] Mixed inline and indented (inline wins)
  [P1174:BACKEND] Should use P1174 from inline
  organization:: OVERRIDE-ME
  ctx:: OVERRIDE-ME-TOO

## Mixed Syntax Priority Test

- [ ] [P1174] org:CLIENT ctx:BACKEND
  Should use: org=P1174 (from []), ctx=BACKEND (from ctx:)

- [ ] org:DANORAMA [P1174:FINANCE]
  Should use: org=P1174 (from []), ctx=FINANCE (from [])

## Override Frontmatter Test

- [ ] [OVERRIDE] This should use OVERRIDE not P1174
- [ ] ctx:FRONTEND This should use FRONTEND not BACKEND
- [ ] proj:NewProject This should use NewProject not Cloud-Migration

## Real-World Example Tasks

- [ ] [P1174:BACKEND] ⏫ Deploy new API endpoint #urgent #backend 📅 2025-11-15 @john
- [ ] [DANORAMA:HR] Schedule interviews for new position #recruiting
- [ ] org:P1174 ctx:FINANCE Monthly invoice review 📅 2025-11-30
- [ ] [INTERNAL] Update flip documentation #documentation
- [ ] Personal fitness goal org:PERSONAL ctx:HEALTH

## Edge Cases

- [ ] Empty brackets [] should not break anything
- [ ] Multiple brackets [ORG1] [ORG2] should use first one
- [ ] ALLCAPS [TEST-123] with-hyphens org:CAPS-OK
- [ ] Numbers [P1174] [123] [CLIENT-456]
