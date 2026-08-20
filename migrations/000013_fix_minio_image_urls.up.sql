UPDATE events
SET image_url = REPLACE(image_url, 'http://localhost:9002', 'https://ticket.bagusbimawan.com')
WHERE image_url LIKE 'http://localhost:9002%';
