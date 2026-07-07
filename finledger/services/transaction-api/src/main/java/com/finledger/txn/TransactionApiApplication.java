package com.finledger.txn;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * FinLedger Transaction API.
 *
 * Accepts money-transfer requests and records them as atomic, double-entry
 * ledger postings (a debit and a credit that must balance). The transfer is
 * wrapped in a single database transaction (@Transactional) so it is all-or-
 * nothing: a partial write can never occur. The ledger is append-only and
 * auditable. After a successful transfer, an event is emitted for the async
 * fraud-check worker.
 *
 * Exposes:
 *   POST /api/transactions          - submit a transfer
 *   GET  /api/transactions/{id}     - look up a transfer
 *   GET  /api/accounts/{id}/balance - account balance (sum of ledger entries)
 *   GET  /actuator/health           - liveness/readiness (Spring Actuator)
 */
@SpringBootApplication
public class TransactionApiApplication {
    public static void main(String[] args) {
        SpringApplication.run(TransactionApiApplication.class, args);
    }
}
