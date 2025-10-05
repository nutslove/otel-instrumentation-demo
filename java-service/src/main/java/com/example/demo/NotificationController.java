package com.example.demo;

import io.opentelemetry.api.trace.Span;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.web.bind.annotation.*;

import java.sql.Timestamp;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@RestController
@CrossOrigin(origins = "*")
public class NotificationController {

    private static final Logger logger = LoggerFactory.getLogger(NotificationController.class);

    @Autowired
    private JdbcTemplate jdbcTemplate;

    @GetMapping("/")
    public Map<String, String> root() {
        Span span = Span.current();
        String traceId = span.getSpanContext().getTraceId();

        logger.info("Java service root endpoint called - trace_id: {}", traceId);

        Map<String, String> response = new HashMap<>();
        response.put("service", "java-spring-boot");
        response.put("status", "running");
        response.put("trace_id", traceId);
        return response;
    }

    @PostMapping("/notifications/send")
    public Map<String, Object> sendNotification(@RequestBody Map<String, Object> request) {
        Span span = Span.current();
        String traceId = span.getSpanContext().getTraceId();

        String recipient = (String) request.get("recipient");
        String message = (String) request.get("message");
        String type = (String) request.get("type");

        logger.info("Sending notification to {} - trace_id: {}", recipient, traceId);

        jdbcTemplate.update(
                "INSERT INTO notifications (recipient, message, type, status, sent_at) VALUES (?, ?, ?, ?, ?)",
                recipient, message, type, "sent", new Timestamp(System.currentTimeMillis())
        );

        logger.info("Notification sent successfully - trace_id: {}", traceId);

        Map<String, Object> response = new HashMap<>();
        response.put("status", "sent");
        response.put("recipient", recipient);
        response.put("type", type);
        response.put("trace_id", traceId);
        return response;
    }

    @GetMapping("/notifications")
    public Map<String, Object> getNotifications() {
        Span span = Span.current();
        String traceId = span.getSpanContext().getTraceId();

        logger.info("Fetching all notifications - trace_id: {}", traceId);

        List<Map<String, Object>> notifications = jdbcTemplate.queryForList(
                "SELECT * FROM notifications ORDER BY sent_at DESC"
        );

        logger.info("Retrieved {} notifications - trace_id: {}", notifications.size(), traceId);

        Map<String, Object> response = new HashMap<>();
        response.put("notifications", notifications);
        response.put("trace_id", traceId);
        return response;
    }

    @GetMapping("/health")
    public Map<String, String> health() {
        Map<String, String> response = new HashMap<>();
        response.put("status", "healthy");
        return response;
    }

    @GetMapping("/error")
    public Map<String, String> intentionalError() {
        Span span = Span.current();
        String traceId = span.getSpanContext().getTraceId();

        logger.error("Intentional error triggered - trace_id: {}", traceId);

        Map<String, String> response = new HashMap<>();
        response.put("error", "Intentional error for testing");
        response.put("trace_id", traceId);

        throw new RuntimeException("Intentional error for testing");
    }
}
