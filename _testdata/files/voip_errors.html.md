Переглянути джерело

**Помилки, які зустрічаються найчастіше:**

* **401 Unauthorized**, неправильно вказаний логін чи пароль.
* **400 Bad Request, **помилка SIP протоколу, будь ласка, зверніться до технічної підтримки.
* **403 Invalid Phone Number**, неправильно вказаний номер абонента.
* **404 Not found**, номер не знайдений на платформі або по номеру немає реєстрації.
* **407 Proxy Authentication Required**, необхідно вказати Proxy-Authenticate або IP адреса не зареєстрована за номером який ви використовуєте (тільки у випадку авторизація за IP адресою).
* **409 Not enough money**,на балансах номера та контракта закінчились гроші.



* **412 Invalid Number**, дзвінок з номера, який вказаний у полі X-FWD-ORIGINAL не проходив через SIP транк lifecell.
* **480 Temporarily Unavailable**, немає реєстрації, не налаштована маршрутизація, номер тимчасово недоступний по різним причинам: не активний, заблокований за несплату, не в мережі тощо.
* **481 Call/Transaction Does Not Exist**, Одна з можливих причин цієї помилки - це невірне значення RAck в пакеті. Наприклад, інвайт мав RAck: 500 364 INVITE, то і наступна відповідь PRACK має містити 364. Якщо вказати інше значення, сервер вважає відповідь некоректною та закінчує дзвінок помилкою
* **486 Busy Here,** абонент Б відхиляє дзвінок АБО під час додзвону абоненту Б зайнятий іншим дзвінком АБО абонент Б встановив заборону вхідних.
* **487 Request Terminated**, завершення дзвінка пакетами bye або cancel, не є помилкою.
* **488 Not Acceptable Here**, шифрування RTP/SRTP дзвінка не співпадає з налаштуванням номеру. Необхідно змінити параметр encryption.
* **500 Service Unavailable**, SIP сервіс недоступний.
* **500 Service Unavailable**. **X-FWD-Original incorrect**, некоректно вказана переадресація дзвінка.
* **501 Not Implemented**, **502 Bad Gateway**, даний напрямок наразі недоступний, будь ласка, зверніться до технічної підтримки.
* **503 Simultaneous calls limit reached**, перевищено ліміт одночасних викликів.
* **603 Declined**, абонент Б відхилив дзвінок.
* **607 Unwanted,** номер Б не бажає прийняти дзвінок, вірогідніше всього ввімкнене локування дзвінків з номера А.




**Як зняти дамп дзвінка**

Зняти дамп дзвінка ви можете будь-яким зручним способом. Нижче декілька популярних варіантів:


Мережева утіліта**[sngrep](http://voiplab.by/wiki/new-voip-technology/105-sngrep-analiz-sip-trafika-s-peredovymi-vozmozhnostyami)**

Програма[Wireshark](https://voxlink.ru/kb/voip-devices-configuration/wireshark-sohranyaem-dump-nuzhnogo-razgovora/)

Будь ласка зберігайте файл у форматі pcap.
















