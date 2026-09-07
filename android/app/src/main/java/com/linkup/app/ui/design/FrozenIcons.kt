package com.linkup.app.ui.design

import androidx.compose.foundation.Canvas
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.unit.dp

enum class FrozenIconKind { PULSE, MAP, FLY, ME, BELL, SEARCH }

@Composable
fun FrozenLineIcon(kind: FrozenIconKind, color: Color, modifier: Modifier = Modifier) {
    Canvas(modifier) {
        val w = size.width
        val h = size.height
        val sw = 2.dp.toPx()
        val stroke = Stroke(width = sw, cap = StrokeCap.Round)
        when (kind) {
            FrozenIconKind.PULSE -> {
                val p = Path().apply {
                    moveTo(w*.55f, h*.08f); lineTo(w*.20f, h*.55f); lineTo(w*.48f, h*.55f)
                    lineTo(w*.40f, h*.92f); lineTo(w*.80f, h*.42f); lineTo(w*.52f, h*.42f); close()
                }
                drawPath(p, color, style = stroke)
            }
            FrozenIconKind.MAP -> {
                drawCircle(color, radius = w*.34f, center = Offset(w*.5f, h*.40f), style = stroke)
                drawCircle(color, radius = w*.08f, center = Offset(w*.5f, h*.40f), style = stroke)
                drawLine(color, Offset(w*.30f,h*.66f), Offset(w*.5f,h*.92f), strokeWidth = sw, cap = StrokeCap.Round)
                drawLine(color, Offset(w*.70f,h*.66f), Offset(w*.5f,h*.92f), strokeWidth = sw, cap = StrokeCap.Round)
            }
            FrozenIconKind.FLY -> {
                val p = Path().apply {
                    moveTo(w*.50f,h*.10f); cubicTo(w*.73f,h*.22f,w*.82f,h*.48f,w*.67f,h*.68f)
                    lineTo(w*.52f,h*.57f); lineTo(w*.33f,h*.75f); lineTo(w*.26f,h*.68f)
                    lineTo(w*.43f,h*.49f); lineTo(w*.32f,h*.34f); cubicTo(w*.38f,h*.22f,w*.45f,h*.14f,w*.50f,h*.10f); close()
                }
                drawPath(p,color,style=stroke)
                drawCircle(color,radius=w*.07f,center=Offset(w*.55f,h*.33f),style=stroke)
            }
            FrozenIconKind.ME -> {
                drawCircle(color, radius = w*.15f, center = Offset(w*.5f,h*.30f), style = stroke)
                val p=Path().apply { moveTo(w*.22f,h*.85f); cubicTo(w*.25f,h*.58f,w*.75f,h*.58f,w*.78f,h*.85f) }
                drawPath(p,color,style=stroke)
            }
            FrozenIconKind.BELL -> {
                val p=Path().apply { moveTo(w*.30f,h*.68f); lineTo(w*.34f,h*.58f); lineTo(w*.34f,h*.38f); cubicTo(w*.34f,h*.17f,w*.66f,h*.17f,w*.66f,h*.38f); lineTo(w*.66f,h*.58f); lineTo(w*.70f,h*.68f); close() }
                drawPath(p,color,style=stroke); drawLine(color,Offset(w*.43f,h*.78f),Offset(w*.57f,h*.78f),strokeWidth=sw,cap=StrokeCap.Round)
            }
            FrozenIconKind.SEARCH -> {
                drawCircle(color,radius=w*.25f,center=Offset(w*.42f,h*.42f),style=stroke)
                drawLine(color,Offset(w*.60f,h*.60f),Offset(w*.84f,h*.84f),strokeWidth=sw,cap=StrokeCap.Round)
            }
        }
    }
}
