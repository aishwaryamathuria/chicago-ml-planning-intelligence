# model_creator.py

import pandas as pd
import psycopg2
from datetime import datetime
from sklearn.model_selection import train_test_split
from sklearn.ensemble import RandomForestRegressor
from sklearn.metrics import mean_squared_error
import pickle

# Connect to PostgreSQL database
conn = psycopg2.connect(
    host="localhost",
    dbname="ChicagoBusinessIntelligence_GOLD",
    user="postgres",
    password="root",
    port="5432"
)

# Load COVID case data
covid_query = """
SELECT zip_code, start_date, end_date, case_count, positive_test_percent
FROM covid_cases
"""
covid_df = pd.read_sql(covid_query, conn)

# Convert weekly data to daily
covid_daily = []
for _, row in covid_df.iterrows():
    days = (row['end_date'] - row['start_date']).days + 1
    if days <= 0 or pd.isna(row['case_count']):
        continue
    weekly_cases = int(row['case_count'])
    base = weekly_cases // days
    remainder = weekly_cases % days

    for i in range(days):
        daily_count = base + (1 if i < remainder else 0)
        covid_daily.append({
            'zip_code': row['zip_code'],
            'date': row['start_date'] + pd.Timedelta(days=i),
            'daily_case_count': daily_count,
            'positive_test_percent': row['positive_test_percent']
        })

covid_daily_df = pd.DataFrame(covid_daily)

# Feature engineering
covid_daily_df['date_ordinal'] = covid_daily_df['date'].apply(lambda x: x.toordinal())
covid_daily_df.rename(columns={'zip_code': 'zip_code_num'}, inplace=True)

# Features and target
features = covid_daily_df[['zip_code_num', 'date_ordinal']]
target = covid_daily_df['daily_case_count']

# Train/test split
X_train, X_test, y_train, y_test = train_test_split(
    features, target, test_size=0.2, random_state=42
)

# Train model
model = RandomForestRegressor(n_estimators=100, random_state=42)
model.fit(X_train, y_train)

# Evaluate
y_pred = model.predict(X_test)
rmse = mean_squared_error(y_test, y_pred, squared=False)
print(f'RMSE: {rmse:.2f}')

# Save model
with open("rf_covid_model_clean.pkl", "wb") as f_model:
    pickle.dump(model, f_model)

# Save test data with predictions
X_test = X_test.copy()
X_test["actual_case_count"] = y_test.values
X_test["predicted_case_count"] = y_pred

print("Model trained and saved to 'rf_covid_model_clean.pkl'")
