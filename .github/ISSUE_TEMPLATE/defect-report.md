---
name: Defect report
about: Something this service did that it should not have, or did not do that it should
title: ""
labels: bug
assignees: ""
---

A report about a media service without the deployment shape, the version, the
network path and what was expected against what happened is an invitation to a
conversation rather than a report. The sections below are what turn it into one.
Fill in what you can and say what you could not find out.

## The version

The line the service prints about itself, pasted rather than described. It names
the revision the binary was built from, and a report against an unknown revision
can only be guessed at.

## The deployment shape

How this is running. Container or binary, the operating system and its version,
the machine, and whether anything sits in front of the service.

Also whether you changed anything from the defaults, and what.

## The network path

Where the participants were relative to the host. Same network, different
networks, behind a home router, on a mobile connection. Which of the ports in the
network shape record you forwarded, and on which protocols.

If media did not flow at all, the startup output and the diagnostic the service
prints are more useful than a description of the symptom.

## What you expected, and what happened

Two statements rather than one. The second is what you observed, in the order you
observed it.

## How to reproduce it

The shortest sequence you know of that produces it, and how often it produces it.
An intermittent defect is still a defect, and how intermittent is part of the
report.

## Logs and output

Paste what the service printed, rather than a summary of it.

This service is built so that its logs carry no conversation content, and a
report is a place that promise can be broken from the other side. Check what you
are pasting for anything about a person before you paste it, and remove it if it
is there.
