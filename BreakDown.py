import numpy as np
import csv

par = []
scores = []

with open('ScoreCard.csv', 'r') as file:
    reader = csv.reader(file)
    header = next(reader)
    for row in reader:
        if row[0].lower() == "par":
            par = list(map(int, row[1:])) 
        else:
            scores.append({"name": row[0], "scores": list(map(int, row[1:]))})

pFront9 = sum(par[0:9])
pBack9 = sum(par[9:18])

front9 = []
back9 = []
totals = []

holes = {"Birdies" : [0]*len(scores), "Pars" : [0]*len(scores), "Bogeys" : [0]*len(scores), "Doubles" : [0]*len(scores), "Worse" : [0]*len(scores)}

for (i, player) in enumerate(scores):
    score = player["scores"]
    total = np.array([s - p for (s, p) in zip(score, par)])
    holes["Birdies"][i] = np.sum(total == -1)
    holes["Pars"][i] = np.sum(total == 0)
    holes["Bogeys"][i] = np.sum(total == 1)
    holes["Doubles"][i] = np.sum(total == 2)
    holes["Worse"][i] = np.sum(total > 2)
    front9.append(np.sum(score[0:9]))
    back9.append(np.sum(score[9:18]))
    totals.append(np.sum(score))

for i, player in enumerate(scores):
    print("========================================")
    print(f"Golfer: {player['name']}")
    for key, value in holes.items():
        print(f"{key}: {value[i]}")
    print("----------------------------------------")
    print(f"Front 9: par: {pFront9} Total: {front9[i]} Scored: {front9[i] - pFront9}")
    print(f"Back 9:  par: {pBack9} Total: {back9[i]} Scored: {back9[i] - pBack9}")

    parScore = sum(par)
    scored = totals[i] - parScore
    print(f"Thru 18: par: {parScore} Total: {totals[i]} Scored: {scored}")

    