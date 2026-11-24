package io.mimir.database;

import com.zaxxer.hikari.HikariDataSource;
import javax.sql.DataSource;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.jdbc.DataSourceProperties;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Primary;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.jdbc.datasource.DataSourceTransactionManager;
import org.springframework.transaction.PlatformTransactionManager;

@Configuration(proxyBeanMethods = false)
public class DatabaseConfiguration {
    @Bean
    @Primary
    @ConfigurationProperties("spring.datasource")
    DataSourceProperties metadataDataSourceProperties() {
        return new DataSourceProperties();
    }

    @Bean(name = "dataSource")
    @Primary
    @ConfigurationProperties("spring.datasource.hikari")
    HikariDataSource metadataDataSource(
            @Qualifier("metadataDataSourceProperties") DataSourceProperties properties) {
        return properties.initializeDataSourceBuilder().type(HikariDataSource.class).build();
    }

    @Bean
    @ConfigurationProperties("mimir.loaded-data.datasource")
    DataSourceProperties loadedDataDataSourceProperties() {
        return new DataSourceProperties();
    }

    @Bean(name = "loadedDataDataSource")
    @ConfigurationProperties("mimir.loaded-data.datasource.hikari")
    HikariDataSource loadedDataDataSource(
            @Qualifier("loadedDataDataSourceProperties") DataSourceProperties properties) {
        return properties.initializeDataSourceBuilder().type(HikariDataSource.class).build();
    }

    @Bean
    @Primary
    JdbcTemplate metadataJdbcTemplate(@Qualifier("dataSource") DataSource dataSource) {
        return new JdbcTemplate(dataSource);
    }

    @Bean
    JdbcTemplate loadedDataJdbcTemplate(@Qualifier("loadedDataDataSource") DataSource dataSource) {
        return new JdbcTemplate(dataSource);
    }

    @Bean
    @Primary
    PlatformTransactionManager transactionManager(@Qualifier("dataSource") DataSource dataSource) {
        return new DataSourceTransactionManager(dataSource);
    }

    @Bean(name = "loadedDataTransactionManager")
    PlatformTransactionManager loadedDataTransactionManager(
            @Qualifier("loadedDataDataSource") DataSource dataSource) {
        return new DataSourceTransactionManager(dataSource);
    }
}
