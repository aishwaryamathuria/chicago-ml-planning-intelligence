import psycopg2
from datetime import datetime, timedelta

BRONZE_DATABASE = {
    'dbname': 'ChicagoBusinessIntelligence_BRONZE',
    'user': 'postgres',
    'password': 'root',
    'host': 'localhost',
    'port': 5432
}

SILVER_DATABASE = {
    'dbname': 'ChicagoBusinessIntelligence_SILVER',
    'user': 'postgres',
    'password': 'root',
    'host': 'localhost',
    'port': 5432
}

TABLE_NAME = "building_permits"

if __name__ == "__main__":
    try:
        conn1 = psycopg2.connect(**BRONZE_DATABASE)
        conn2 = psycopg2.connect(**SILVER_DATABASE)
        cur1 = conn1.cursor()
        cur2 = conn2.cursor()

        # Fetch rows from past 12 hours
        time_diff = datetime.now() - timedelta(hours=12)
        query = f"SELECT * FROM {TABLE_NAME} WHERE last_updated >= '{time_diff}'"
        cur1.execute(query)
        rows = cur1.fetchall()

        # Insert into silver DB
        for row in rows:
            placeholders = ','.join(['%s'] * len(row))
            cur2.execute(f"INSERT INTO {TABLE_NAME} VALUES ({placeholders})", row)
        conn2.commit()

        cur2.execute(f"""
            UPDATE {TABLE_NAME}
            SET permit_status = NULL
            WHERE permit_status IS NULL OR permit_status = '[null]';
        """)
        conn2.commit()

        cur2.execute(f"""
            UPDATE {TABLE_NAME}
            SET community_area = NULL
            WHERE community_area IS NULL OR community_area = '[null]';
        """)
        conn2.commit()

        cur2.execute(f"""
            UPDATE {TABLE_NAME}
            SET location = NULL
            WHERE location IN ('POINT(0, 0)','POINT(nan, nan)', 'POINT(1, 1)', '[null]');
        """)
        conn2.commit()

    except Exception as e:
        print("Error occurred:", e)

    finally:
        cur1.close()
        cur2.close()
        conn1.close()
        conn2.close()
