package com.panelpulse.aggregator;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.CommandLineRunner;
import org.springframework.context.annotation.Bean;
import javax.sql.DataSource;
import java.sql.Connection;
import java.sql.Statement;

/**
 * PanelPulse Aggregation Job.
 *
 * A RUN-TO-COMPLETION batch job (run by a Kubernetes CronJob). It rolls the
 * processed exposure events into measurement aggregates that downstream
 * reporting (P4 RatingsBoard) consumes:
 *   - impressions per channel/daypart
 *   - total viewing minutes
 *   - reach (distinct panelists)
 *
 * Computes the aggregates with SQL against the results store and writes them to
 * an aggregates table, then exits. Idempotent per run window.
 *
 * Original, generic reference design — not based on any proprietary system.
 */
@SpringBootApplication
public class AggregatorApplication {

    public static void main(String[] args) {
        System.exit(SpringApplication.exit(SpringApplication.run(AggregatorApplication.class, args)));
    }

    @Bean
    public CommandLineRunner aggregate(DataSource ds) {
        return args -> {
            try (Connection c = ds.getConnection(); Statement st = c.createStatement()) {
                // Ensure the aggregates table exists.
                st.execute("""
                    CREATE TABLE IF NOT EXISTS measurement_aggregates (
                      channel_id   TEXT NOT NULL,
                      daypart      TEXT NOT NULL,
                      impressions  BIGINT NOT NULL,
                      viewing_min  BIGINT NOT NULL,
                      reach        BIGINT NOT NULL,
                      computed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
                      PRIMARY KEY (channel_id, daypart)
                    )""");

                // Roll up processed_events into aggregates (upsert).
                // impressions = event count; viewing_min = sum(duration)/60;
                // reach = distinct panelists.
                int rows = st.executeUpdate("""
                    INSERT INTO measurement_aggregates (channel_id, daypart, impressions, viewing_min, reach)
                    SELECT channel_id, daypart,
                           COUNT(*) AS impressions,
                           (SUM(duration_sec) / 60) AS viewing_min,
                           COUNT(DISTINCT panelist_id) AS reach
                    FROM processed_events
                    GROUP BY channel_id, daypart
                    ON CONFLICT (channel_id, daypart) DO UPDATE SET
                       impressions = EXCLUDED.impressions,
                       viewing_min = EXCLUDED.viewing_min,
                       reach = EXCLUDED.reach,
                       computed_at = now()
                    """);
                System.out.println("aggregation complete: " + rows + " channel/daypart aggregates updated");
            }
        };
    }
}
