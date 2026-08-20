UPDATE events
SET image_url = REPLACE(image_url, 'https://ticket.bagusbimawan.com', 'http://localhost:9002')
WHERE image_url LIKE 'https://ticket.bagusbimawan.com/event-images%';
