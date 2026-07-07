package com.panelpulse.processor;

import java.time.Instant;

/** Cleaned, enriched event written to the results store. */
public class ProcessedEvent {
    public String panelistId;
    public String channelId;
    public int durationSec;
    public Instant startedAt;
    public String daypart;
    public Instant processedAt;
}
