SET SESSION group_concat_max_len = 10485760;
SELECT SUBSTRING_INDEX(SUBSTRING_INDEX(GROUP_CONCAT(ROUND(timer_wait/1000000) ORDER BY ROUND(timer_wait/1000000) DESC),',' , -ROUND(0.95*COUNT(CURRENT_SCHEMA))), ',', 1 ) AS P95 
FROM performance_schema.events_statements_history 
WHERE LEFT(DIGEST_TEXT,6) IN ('SELECT','INSERT','UPDATE','DELETE','COMMIT') 
AND CURRENT_SCHEMA NOT IN ('performance_schema','information_schema','mysql','sys') 
AND CURRENT_SCHEMA IS NOT NULL;
