package com.finledger.txn;

import jakarta.validation.Valid;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import java.math.BigDecimal;
import java.util.Map;

@RestController
@RequestMapping("/api")
public class TransactionController {

    private final LedgerService ledger;
    private final TransferRepository transfers;

    public TransactionController(LedgerService ledger, TransferRepository transfers) {
        this.ledger = ledger;
        this.transfers = transfers;
    }

    @PostMapping("/transactions")
    public ResponseEntity<Transfer> create(@Valid @RequestBody TransferRequest req) {
        Transfer t = ledger.transfer(req.fromAccount, req.toAccount, req.amount);
        // emitEvent(t) -> async fraud worker (RabbitMQ/Kafka); logged stub here.
        System.out.println("event transaction.created id=" + t.getId()
            + " amount=" + t.getAmount());
        return ResponseEntity.status(HttpStatus.CREATED).body(t);
    }

    @GetMapping("/transactions/{id}")
    public ResponseEntity<Transfer> get(@PathVariable Long id) {
        return transfers.findById(id)
            .map(ResponseEntity::ok)
            .orElse(ResponseEntity.notFound().build());
    }

    @GetMapping("/accounts/{id}/balance")
    public Map<String, Object> balance(@PathVariable Long id) {
        BigDecimal bal = ledger.balance(id);
        return Map.of("account", id, "balance", bal);
    }
}
