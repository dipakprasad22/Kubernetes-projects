package com.panelpulse.processor;

import java.time.Instant;

/** Raw exposure event as consumed from Kafka. */
public class RawEvent {
    public String panelistId;
    public String channelId;
    public Instant startedAt;
    public int durationSec;
}
