package com.panelpulse.processor;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * PanelPulse Stream Processor.
 *
 * Consumes raw exposure events from Kafka, then validates, enriches, and
 * deduplicates them, writing clean records to the results store. Runs as a
 * Deployment scaled by an HPA — under load (growing consumer lag / backlog),
 * more processor pods are added to clear the backlog faster. Kafka's partitions
 * allow parallel consumption across pods.
 *
 * Exposes /actuator/health for K8s probes and consumer-lag-style metrics for HPA.
 *
 * Original, generic reference design — not based on any proprietary system.
 */
@SpringBootApplication
public class ProcessorApplication {
    public static void main(String[] args) {
        SpringApplication.run(ProcessorApplication.class, args);
    }
}
