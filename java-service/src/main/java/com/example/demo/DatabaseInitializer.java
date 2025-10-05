package com.example.demo;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.CommandLineRunner;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Component;

@Component
public class DatabaseInitializer implements CommandLineRunner {

    @Autowired
    private JdbcTemplate jdbcTemplate;

    @Override
    public void run(String... args) {
        jdbcTemplate.execute(
            "CREATE TABLE IF NOT EXISTS notifications (" +
            "id INTEGER PRIMARY KEY AUTOINCREMENT, " +
            "recipient TEXT NOT NULL, " +
            "message TEXT NOT NULL, " +
            "type TEXT NOT NULL, " +
            "status TEXT NOT NULL, " +
            "sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"
        );
    }
}
