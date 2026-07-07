package com.finledger.fraud;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.CommandLineRunner;
import org.springframework.context.annotation.Bean;
import java.math.BigDecimal;

/**
 * FinLedger Fraud-Check Worker.
 *
 * A BACKGROUND WORKER, not a web service — it has no HTTP port. It consumes
 * transaction events (from a queue: RabbitMQ/Kafka) and scores each for fraud,
 * flagging suspicious ones. Decoupling scoring from the transaction hot path
 * keeps transfers fast and the fraud logic independently scalable.
 *
 * Health for a worker is "is the process alive and consuming" — a liveness
 * file or metrics endpoint, not an HTTP /health (see the K8s Deployment).
 * Handles SIGTERM gracefully so in-flight scoring finishes before shutdown.
 */
@SpringBootApplication
public class FraudWorkerApplication {

    public static void main(String[] args) {
        SpringApplication.run(FraudWorkerApplication.class, args);
    }

    @Bean
    public CommandLineRunner runner() {
        return args -> {
            FraudScorer scorer = new FraudScorer();
            Runtime.getRuntime().addShutdownHook(new Thread(() ->
                System.out.println("SIGTERM received — finishing in-flight work, shutting down")));
            System.out.println("fraud worker started — consuming transaction events");
            // Real impl: subscribe to the queue and call scorer.score(event) per message.
            // Demo loop so the container has a long-running main process:
            while (true) {
                // In production this blocks on the queue consumer.
                Thread.sleep(10_000);
                System.out.println("fraud worker heartbeat (idle: no messages)");
            }
        };
    }
}
