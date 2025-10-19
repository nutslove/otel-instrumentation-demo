package com.example.demo;

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
        logger.info("Java service root endpoint called");

        Map<String, String> response = new HashMap<>();
        response.put("service", "java-spring-boot");
        response.put("status", "running");
        return response;
    }

    @PostMapping("/notifications/send")
    public Map<String, Object> sendNotification(@RequestBody Map<String, Object> request) {
        String recipient = (String) request.get("recipient");
        String message = (String) request.get("message");
        String type = (String) request.get("type");

        logger.info("Sending notification to {}", recipient);

        jdbcTemplate.update(
                "INSERT INTO notifications (recipient, message, type, status, sent_at) VALUES (?, ?, ?, ?, ?)",
                recipient, message, type, "sent", new Timestamp(System.currentTimeMillis())
        );

        logger.info("Notification sent successfully");

        Map<String, Object> response = new HashMap<>();
        response.put("status", "sent");
        response.put("recipient", recipient);
        response.put("type", type);
        return response;
    }

    @GetMapping("/notifications")
    public Map<String, Object> getNotifications() {
        logger.info("Fetching all notifications");

        List<Map<String, Object>> notifications = jdbcTemplate.queryForList(
                "SELECT * FROM notifications ORDER BY sent_at DESC"
        );

        logger.info("Retrieved {} notifications", notifications.size());

        Map<String, Object> response = new HashMap<>();
        response.put("notifications", notifications);
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
        logger.error("Intentional error triggered");

        Map<String, String> response = new HashMap<>();
        response.put("error", "Intentional error for testing");

        throw new RuntimeException("Intentional error for testing");
    }
}
