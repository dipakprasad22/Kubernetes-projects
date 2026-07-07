package com.panelpulse.processor;

import org.springframework.stereotype.Component;
import java.time.Instant;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Core stream-processing logic for one exposure event. Pure and testable —
 * the Kafka consumer wiring calls process() per message.
 */
@Component
public class EventProcessor {

    // Dedup window: remember recently-seen event keys to drop duplicates
    // (panel meters can resend). Bounded in-memory set (real systems use a
    // store/RocksDB/Kafka Streams state); kept simple here.
    private final Set<String> seen = ConcurrentHashMap.newKeySet();

    public ProcessedEvent process(RawEvent raw) {
        // 1. Validate — reject malformed events (defensive; collector did light checks).
        if (raw.panelistId == null || raw.channelId == null || raw.durationSec <= 0) {
            throw new InvalidEventException("missing required fields");
        }
        // Cap absurd durations (a single exposure > 24h is bad data).
        if (raw.durationSec > 86_400) {
            throw new InvalidEventException("duration too large: " + raw.durationSec);
        }

        // 2. Deduplicate — drop if we've seen this exact event key recently.
        String key = raw.panelistId + "|" + raw.channelId + "|" + raw.startedAt;
        if (!seen.add(key)) {
            throw new DuplicateEventException(key);
        }
        if (seen.size() > 1_000_000) seen.clear(); // crude bound for the demo

        // 3. Enrich — add derived fields used by aggregation (e.g. daypart).
        ProcessedEvent out = new ProcessedEvent();
        out.panelistId = raw.panelistId;
        out.channelId = raw.channelId;
        out.durationSec = raw.durationSec;
        out.startedAt = raw.startedAt;
        out.daypart = daypartOf(raw.startedAt);
        out.processedAt = Instant.now();
        return out;
    }

    // Derive a daypart bucket (a standard measurement enrichment).
    static String daypartOf(Instant startedAt) {
        int hour = startedAt.atZone(java.time.ZoneOffset.UTC).getHour();
        if (hour < 6) return "overnight";
        if (hour < 10) return "morning";
        if (hour < 16) return "daytime";
        if (hour < 19) return "early_fringe";
        if (hour < 23) return "primetime";
        return "late_fringe";
    }
}
