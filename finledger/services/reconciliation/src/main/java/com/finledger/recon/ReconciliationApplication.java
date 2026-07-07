package com.finledger.recon;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.CommandLineRunner;
import org.springframework.beans.factory.annotation.Value;
import javax.sql.DataSource;
import java.math.BigDecimal;
import java.sql.Connection;
import java.sql.ResultSet;
import java.sql.Statement;

/**
 * FinLedger Reconciliation Job.
 *
 * A RUN-TO-COMPLETION batch job (invoked by a Kubernetes CronJob on a schedule).
 * It verifies the core ledger invariant: the sum of ALL signed ledger entries
 * must be exactly zero (every debit has an equal credit). If it isn't, the
 * ledger is out of balance — the job exits non-zero so the CronJob records a
 * failure and alerting can fire.
 *
 * Unlike the API and worker, this is NOT a long-running process: it does its
 * check and exits (System.exit), which is exactly what a Job expects.
 */
@SpringBootApplication
public class ReconciliationApplication {

    public static void main(String[] args) {
        // Don't keep the web server / context alive; run and exit.
        System.exit(SpringApplication.exit(SpringApplication.run(ReconciliationApplication.class, args)));
    }

    @org.springframework.context.annotation.Bean
    public CommandLineRunner reconcile(DataSource ds) {
        return args -> {
            try (Connection c = ds.getConnection(); Statement st = c.createStatement()) {
                ResultSet rs = st.executeQuery(
                    "SELECT COALESCE(SUM(amount), 0) AS total FROM ledger_entries");
                BigDecimal total = BigDecimal.ZERO;
                if (rs.next()) total = rs.getBigDecimal("total");

                System.out.println("reconciliation: ledger total = " + total);
                if (total.compareTo(BigDecimal.ZERO) != 0) {
                    System.err.println("RECONCILIATION FAILED: ledger out of balance by " + total);
                    throw new IllegalStateException("ledger imbalance: " + total);
                }
                System.out.println("reconciliation OK: ledger balanced (sum == 0)");
            }
        };
    }
}
