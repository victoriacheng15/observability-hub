#!/bin/bash
cd /home/server/software/observability-platform/system-metrics

./metrics-collector.exe >> /var/log/system-metrics.log 2>&1