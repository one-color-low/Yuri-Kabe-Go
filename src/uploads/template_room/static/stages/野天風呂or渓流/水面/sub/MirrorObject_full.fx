////////////////////////////////////////////////////////////////////////////////////////////////
//
//  MWF_Object.fx ver0.0.2  ���f���������`��
//  ( MirrorWF.fx ����Ăяo����܂��D�I�t�X�N���[���`��p)
//  �쐬: �j��P( ���͉��P����Mirror.fx, full.fx,���� )
//
////////////////////////////////////////////////////////////////////////////////////////////////
// �A�N�Z�ɑg�ݍ��ޏꍇ�͂�����K�X�ύX���Ă��������D
float3 MirrorPos = float3( 0.0, 0.0, 0.0 );    // ���[�J�����W�n�ɂ����鋾�ʏ�̔C�ӂ̍��W(�A�N�Z���_���W�̈�_)
float3 MirrorNormal = float3( 0.0, 1.0, 0.0 ); // ���[�J�����W�n�ɂ����鋾�ʂ̖@���x�N�g��

///////////////////////////////////////////////////////////////////////////////////////////////

// ���[�J�����W�n�ɂ����鋾���ʒu�ւ̕ϊ�
static float3 n = normalize(MirrorNormal);
static float4x4 MirrorMatrix = { 1.0f-2.0f*n.x*n.x, -2.0f*n.x*n.y,      -2.0f*n.x*n.z,      0.0f,
                                -2.0f*n.x*n.y,       1.0f-2.0f*n.y*n.y, -2.0f*n.y*n.z,      0.0f,
                                -2.0f*n.x*n.z,      -2.0f*n.y*n.z,       1.0f-2.0f*n.z*n.z, 0.0f,
                                 MirrorPos.x,        MirrorPos.y,        MirrorPos.z,       1.0f};
float4 TransMirrorPos( float4 Pos ){
    Pos = float4(Pos.xyz - MirrorPos, 1.0f);
    Pos = mul( Pos, MirrorMatrix ); // �����ϊ�
    return Pos;
}

// ���[���h�ϊ��s�����ŁA�t�s����v�Z����B
// - �s�񂪁A���{�X�P�[�����O�A��]�A���s�ړ������܂܂Ȃ����Ƃ�O������Ƃ���B
float4x4 InverseWorldMatrix(float4x4 mat) {
    float scaling = length(mat._11_12_13);
    float scaling_inv = 1.0 / (scaling * scaling);

    float3x3 mat3x3_inv = transpose((float3x3)mat) * scaling_inv;
    return float4x4( mat3x3_inv[0], 0, 
                     mat3x3_inv[1], 0, 
                     mat3x3_inv[2], 0, 
                     -mul(mat._41_42_43, mat3x3_inv), 1 );
}

// ���ʍ��W�ϊ��p�����[�^
float4x4 MirrorWorldMatrix: CONTROLOBJECT < string Name = "(OffscreenOwner)"; >; // ���ʃA�N�Z�̃��[���h�ϊ��s��
static float4x4 InvMirrorWorldMatrix = InverseWorldMatrix(MirrorWorldMatrix);    // ���ʃA�N�Z�̃��[���h�ϊ��t�s��

// Obj���W�ϊ��s��
float4x4 WorldViewProjMatrix      : WORLDVIEWPROJECTION;
float4x4 WorldMatrix              : WORLD;
float4x4 ViewMatrix               : VIEW;
float4x4 ViewProjMatrix           : VIEWPROJECTION;
float4x4 LightWorldViewProjMatrix : WORLDVIEWPROJECTION < string Object = "Light"; >;

float3   LightDirection    : DIRECTION < string Object = "Light"; >;
float3   CameraPosition    : POSITION  < string Object = "Camera"; >;

// �}�e���A���F
float4   MaterialDiffuse   : DIFFUSE  < string Object = "Geometry"; >;
float3   MaterialAmbient   : AMBIENT  < string Object = "Geometry"; >;
float3   MaterialEmmisive  : EMISSIVE < string Object = "Geometry"; >;
float3   MaterialSpecular  : SPECULAR < string Object = "Geometry"; >;
float    SpecularPower     : SPECULARPOWER < string Object = "Geometry"; >;
float3   MaterialToon      : TOONCOLOR;
float4   EdgeColor         : EDGECOLOR;
// ���C�g�F
float3   LightDiffuse      : DIFFUSE   < string Object = "Light"; >;
float3   LightAmbient      : AMBIENT   < string Object = "Light"; >;
float3   LightSpecular     : SPECULAR  < string Object = "Light"; >;
static float4 DiffuseColor  = MaterialDiffuse  * float4(LightDiffuse, 1.0f);
static float3 AmbientColor  = MaterialAmbient  * LightAmbient + MaterialEmmisive;
static float3 SpecularColor = MaterialSpecular * LightSpecular;

bool     parthf;   // �p�[�X�y�N�e�B�u�t���O
bool     transp;   // �������t���O
bool	 spadd;    // �X�t�B�A�}�b�v���Z�����t���O
#define SKII1    1500
#define SKII2    8000
#define Toon     3

// �I�u�W�F�N�g�̃e�N�X�`��
texture ObjectTexture: MATERIALTEXTURE;
sampler ObjTexSampler = sampler_state {
    texture = <ObjectTexture>;
    MINFILTER = LINEAR;
    MAGFILTER = LINEAR;
};

// �X�t�B�A�}�b�v�̃e�N�X�`��
texture ObjectSphereMap: MATERIALSPHEREMAP;
sampler ObjSphareSampler = sampler_state {
    texture = <ObjectSphereMap>;
    MINFILTER = LINEAR;
    MAGFILTER = LINEAR;
};

// MMD�{����sampler���㏑�����Ȃ����߂̋L�q�ł��B�폜�s�B
sampler MMDSamp0 : register(s0);
sampler MMDSamp1 : register(s1);
sampler MMDSamp2 : register(s2);

////////////////////////////////////////////////////////////////////////////////////////////////
// �֊s�`��

struct VS_OUTPUT0 {
    float4 Pos        : POSITION;    // �ˉe�ϊ����W
    float4 VPos       : TEXCOORD5;   // �������f���̃��[���h���W
    float4 VCamera    : TEXCOORD6;   // �J�����Ƌ��ʂ̑��΍��W
};

// �����G�b�W���_�V�F�[�_
VS_OUTPUT0 MirrorEdge_VS(float4 Pos : POSITION)
{
    VS_OUTPUT0 Out = (VS_OUTPUT0)0;

    // ���[���h���W�ϊ�
    Pos = mul( Pos, WorldMatrix );

    // �����ʒu�ւ̍��W�ϊ�
    Pos = mul( Pos, InvMirrorWorldMatrix );
    Pos = TransMirrorPos( Pos ); // �����ϊ�
    Out.VPos = Pos;
    Pos = mul( Pos, MirrorWorldMatrix );

    // �J�������_�̃r���[�ˉe�ϊ�
    Out.Pos = mul( Pos, ViewProjMatrix );
    Out.Pos.x *= -1.0f; // �|���S�������Ԃ�Ȃ��悤�ɍ��E���]�ɂ��ĕ`��

    // �J�����̋��ʂɑ΂��鑊�΍��W
    Out.VCamera = mul( float4(CameraPosition, 1.0f), InvMirrorWorldMatrix );

    return Out;
}

// �s�N�Z���V�F�[�_
float4 Edge_PS(VS_OUTPUT0 IN) : COLOR
{
    // �֊s�F�œh��Ԃ�
    float4 Color = EdgeColor;

    // ���ʂ̗����ɂ��镔�ʂ͋����\�����Ȃ�
    if(dot((float3)IN.VPos-MirrorPos, MirrorNormal)*dot((float3)IN.VCamera-MirrorPos, MirrorNormal) >= 0.0f) Color.a = 0.0f;

    return Color;
}

// �֊s�`��p�e�N�j�b�N
technique EdgeTec < string MMDPass = "edge"; > {
    pass DrawMirrorEdge {
        AlphaBlendEnable = TRUE;
        AlphaTestEnable  = TRUE;
        SrcBlend = SRCALPHA;
        DestBlend = INVSRCALPHA;
        VertexShader = compile vs_2_0 MirrorEdge_VS();
        PixelShader  = compile ps_2_0 Edge_PS();
    }
}


///////////////////////////////////////////////////////////////////////////////////////////////
// �e�i��Z���t�V���h�E�j�`��

// ���_�V�F�[�_
VS_OUTPUT0 ShadowMirror_VS(float4 Pos : POSITION)
{
    VS_OUTPUT0 Out = (VS_OUTPUT0)0;

    // ���[���h���W�ϊ�
    Pos = mul( Pos, WorldMatrix );

    // �����ʒu�ւ̍��W�ϊ�
    Pos = mul( Pos, InvMirrorWorldMatrix );
    Pos = TransMirrorPos( Pos ); // �����ϊ�
    Out.VPos = Pos;
    Pos = mul( Pos, MirrorWorldMatrix );

    // �J�������_�̃r���[�ˉe�ϊ�
    Out.Pos = mul( Pos, ViewProjMatrix );
    Out.Pos.x *= -1.0f; // �|���S�������Ԃ�Ȃ��悤�ɍ��E���]�ɂ��ĕ`��

    // �J�����̋��ʂɑ΂��鑊�΍��W
    Out.VCamera = mul( float4(CameraPosition, 1.0f), InvMirrorWorldMatrix );

    return Out;
}

// �s�N�Z���V�F�[�_
float4 Shadow_PS(VS_OUTPUT0 IN) : COLOR
{
    // �A���r�G���g�F�œh��Ԃ�
    float4 Color = float4(AmbientColor.rgb, 0.65f);;

    // ���ʂ̗����ɂ��镔�ʂ͋����\�����Ȃ�
    if(dot((float3)IN.VPos-MirrorPos, MirrorNormal)*dot((float3)IN.VCamera-MirrorPos, MirrorNormal) >= 0.0f) Color.a = 0.0f;

    return Color;
}

// �e�`��p�e�N�j�b�N
technique ShadowTec < string MMDPass = "shadow"; > {
    pass DrawShadow {
        VertexShader = compile vs_2_0 ShadowMirror_VS();
        PixelShader  = compile ps_2_0 Shadow_PS();
    }
}


///////////////////////////////////////////////////////////////////////////////////////////////
// �I�u�W�F�N�g�`��i�Z���t�V���h�EOFF�j

struct VS_OUTPUT {
    float4 Pos        : POSITION;    // �ˉe�ϊ����W
    float2 Tex        : TEXCOORD1;   // �e�N�X�`��
    float3 Normal     : TEXCOORD2;   // �@��
    float3 Eye        : TEXCOORD3;   // �J�����Ƃ̑��Έʒu
    float2 SpTex      : TEXCOORD4;   // �X�t�B�A�}�b�v�e�N�X�`�����W
    float4 VPos       : TEXCOORD5;   // �������f���̃��[���h���W
    float4 VCamera    : TEXCOORD6;   // �J�����Ƌ��ʂ̑��΍��W
    float4 Color      : COLOR0;      // �f�B�t���[�Y�F
};

// ���_�V�F�[�_
VS_OUTPUT BasicMirror_VS(float4 Pos : POSITION, float3 Normal : NORMAL, float2 Tex : TEXCOORD0, uniform bool useTexture, uniform bool useSphereMap, uniform bool useToon)
{
    VS_OUTPUT Out = (VS_OUTPUT)0;

    // ���[���h���W�ϊ�
    Pos = mul( Pos, WorldMatrix );

    // �J�����Ƃ̑��Έʒu(����������������Ă��邱�Ƃ��l��)
    Out.Eye = CameraPosition - Pos;

    // �����ʒu�ւ̍��W�ϊ�
    Pos = mul( Pos, InvMirrorWorldMatrix );
    Pos = TransMirrorPos( Pos ); // �����ϊ�
    Out.VPos = Pos;
    Pos = mul( Pos, MirrorWorldMatrix );

    // �J�������_�̃r���[�ˉe�ϊ�
    Out.Pos = mul( Pos, ViewProjMatrix );
    Out.Pos.x *= -1.0f; // �|���S�������Ԃ�Ȃ��悤�ɍ��E���]�ɂ��ĕ`��

    // �J�����̋��ʂɑ΂��鑊�΍��W
    Out.VCamera = mul( float4(CameraPosition, 1.0f), InvMirrorWorldMatrix );

    // ���_�@��(����������������Ă��邱�Ƃ��l��)
    Out.Normal = normalize( mul( Normal, (float3x3)WorldMatrix ) );

    // �f�B�t���[�Y�F�{�A���r�G���g�F �v�Z
    Out.Color.rgb = AmbientColor;
    if ( !useToon ) {
        Out.Color.rgb += max(0,dot( Out.Normal, -LightDirection )) * DiffuseColor.rgb;
    }
    Out.Color.a = DiffuseColor.a;
    Out.Color = saturate( Out.Color );

    // �e�N�X�`�����W
    Out.Tex = abs(frac(Tex)*0.5);

    if ( useSphereMap ) {
        // �X�t�B�A�}�b�v�e�N�X�`�����W
        float2 NormalWV = mul( Out.Normal, (float3x3)ViewMatrix );
        Out.SpTex.x = NormalWV.x * 0.5f + 0.5f;
        Out.SpTex.y = NormalWV.y * -0.5f + 0.5f;
    }

    return Out;
}

// �s�N�Z���V�F�[�_
float4 Basic_PS(VS_OUTPUT IN, uniform bool useTexture, uniform bool useSphereMap, uniform bool useToon) : COLOR0
{
    // �X�y�L�����F�v�Z
    float3 HalfVector = normalize( normalize(IN.Eye) + -LightDirection );
    float3 Specular = pow( max(0,dot( HalfVector, normalize(IN.Normal) )), SpecularPower ) * SpecularColor;

    float4 Color = IN.Color;
    if ( useTexture ) {
        // �e�N�X�`���K�p
        Color *= tex2D( ObjTexSampler, IN.Tex );
    }
    if ( useSphereMap ) {
        // �X�t�B�A�}�b�v�K�p
        if(spadd) Color.rgb += tex2D(ObjSphareSampler,IN.SpTex).rgb;
        else      Color.rgb *= tex2D(ObjSphareSampler,IN.SpTex).rgb;
    }

    if ( useToon ) {
        // �g�D�[���K�p
        float LightNormal = dot( IN.Normal, -LightDirection );
//        Color.rgb *= lerp(MaterialToon, float3(1,1,1), saturate(LightNormal * 16 + 0.5)); // �g�[��bmp�ɂ���Ă͂���ł͍Č��ł��Ȃ��݂����ł�
        Color.rgb *= tex2D( MMDSamp0, float2(0.5f, 0.5f-LightNormal*0.5f) ).rgb;
    }

    // �X�y�L�����K�p
    Color.rgb += Specular;

    // ���ʂ̗����ɂ��镔�ʂ͋����\�����Ȃ�
    if(dot((float3)IN.VPos-MirrorPos, MirrorNormal)*dot((float3)IN.VCamera-MirrorPos, MirrorNormal) >= 0.0f) Color.a = 0.0f;

    return Color;
}

// �I�u�W�F�N�g�`��p�e�N�j�b�N�i�A�N�Z�T���p�j
// �s�v�Ȃ��͍̂폜��
technique MainTec0 < string MMDPass = "object"; bool UseTexture = false; bool UseSphereMap = false; bool UseToon = false; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_2_0 BasicMirror_VS(false, false, false);
        PixelShader  = compile ps_2_0 Basic_PS(false, false, false);
    }
}

technique MainTec1 < string MMDPass = "object"; bool UseTexture = true; bool UseSphereMap = false; bool UseToon = false; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_2_0 BasicMirror_VS(true, false, false);
        PixelShader  = compile ps_2_0 Basic_PS(true, false, false);
    }
}

technique MainTec2 < string MMDPass = "object"; bool UseTexture = false; bool UseSphereMap = true; bool UseToon = false; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_2_0 BasicMirror_VS(false, true, false);
        PixelShader  = compile ps_2_0 Basic_PS(false, true, false);
    }
}

technique MainTec3 < string MMDPass = "object"; bool UseTexture = true; bool UseSphereMap = true; bool UseToon = false; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_2_0 BasicMirror_VS(true, true, false);
        PixelShader  = compile ps_2_0 Basic_PS(true, true, false);
    }
}

// �I�u�W�F�N�g�`��p�e�N�j�b�N�iPMD���f���p�j
technique MainTec4 < string MMDPass = "object";
                     bool UseTexture = false; bool UseSphereMap = false; bool UseToon = true; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_2_0 BasicMirror_VS(false, false, true);
        PixelShader  = compile ps_2_0 Basic_PS(false, false, true);
    }
}

technique MainTec5 < string MMDPass = "object";
                     bool UseTexture = true; bool UseSphereMap = false; bool UseToon = true; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_2_0 BasicMirror_VS(true, false, true);
        PixelShader  = compile ps_2_0 Basic_PS(true, false, true);
    }
}

technique MainTec6 < string MMDPass = "object";
                     bool UseTexture = false; bool UseSphereMap = true; bool UseToon = true; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_2_0 BasicMirror_VS(false, true, true);
        PixelShader  = compile ps_2_0 Basic_PS(false, true, true);
    }
}

technique MainTec7 < string MMDPass = "object";
                     bool UseTexture = true; bool UseSphereMap = true; bool UseToon = true; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_2_0 BasicMirror_VS(true, true, true);
        PixelShader  = compile ps_2_0 Basic_PS(true, true, true);
    }
}


///////////////////////////////////////////////////////////////////////////////////////////////
// �Z���t�V���h�E�pZ�l�v���b�g

struct VS_ZValuePlot_OUTPUT {
    float4 Pos : POSITION;              // �ˉe�ϊ����W
    float4 ShadowMapTex : TEXCOORD0;    // Z�o�b�t�@�e�N�X�`��
};

// ���_�V�F�[�_
VS_ZValuePlot_OUTPUT ZValuePlot_VS( float4 Pos : POSITION )
{
    VS_ZValuePlot_OUTPUT Out = (VS_ZValuePlot_OUTPUT)0;

    // ���C�g�̖ڐ��ɂ�郏�[���h�r���[�ˉe�ϊ�������
    Out.Pos = mul( Pos, LightWorldViewProjMatrix );

    // �e�N�X�`�����W�𒸓_�ɍ��킹��
    Out.ShadowMapTex = Out.Pos;

    return Out;
}

// �s�N�Z���V�F�[�_
float4 ZValuePlot_PS( float4 ShadowMapTex : TEXCOORD0 ) : COLOR
{
    // R�F������Z�l���L�^����
    return float4(ShadowMapTex.z/ShadowMapTex.w,0,0,1);
}

// Z�l�v���b�g�p�e�N�j�b�N
technique ZplotTec < string MMDPass = "zplot"; > {
    pass ZValuePlot {
        AlphaBlendEnable = FALSE;
        VertexShader = compile vs_2_0 ZValuePlot_VS();
        PixelShader  = compile ps_2_0 ZValuePlot_PS();
    }
}


///////////////////////////////////////////////////////////////////////////////////////////////
// �I�u�W�F�N�g�`��i�Z���t�V���h�EON�j

// �V���h�E�o�b�t�@�̃T���v���B"register(s0)"�Ȃ̂�MMD��s0���g���Ă��邩��
sampler DefSampler : register(s0);

struct BufferShadow_OUTPUT {
    float4 Pos      : POSITION;     // �ˉe�ϊ����W
    float4 ZCalcTex : TEXCOORD0;    // Z�l
    float2 Tex      : TEXCOORD1;    // �e�N�X�`��
    float3 Normal   : TEXCOORD2;    // �@��
    float3 Eye      : TEXCOORD3;    // �J�����Ƃ̑��Έʒu
    float2 SpTex    : TEXCOORD4;    // �X�t�B�A�}�b�v�e�N�X�`�����W
    float4 VPos     : TEXCOORD5;    // �������f���̃��[���h���W
    float4 VCamera  : TEXCOORD6;    // �J�����Ƌ��ʂ̑��΍��W
    float4 Color    : COLOR0;       // �f�B�t���[�Y�F
};

// ���_�V�F�[�_
BufferShadow_OUTPUT BufferShadowMirror_VS(float4 Pos : POSITION, float3 Normal : NORMAL, float2 Tex : TEXCOORD0, uniform bool useTexture, uniform bool useSphereMap, uniform bool useToon)
{
    BufferShadow_OUTPUT Out = (BufferShadow_OUTPUT)0;

    // ���C�g���_�ɂ�郏�[���h�r���[�ˉe�ϊ�(����������������Ă��邱�Ƃ��l��)
    Out.ZCalcTex = mul( Pos, LightWorldViewProjMatrix );

    // ���[���h���W�ϊ�
    Pos = mul( Pos, WorldMatrix );

    // �J�����Ƃ̑��Έʒu(����������������Ă��邱�Ƃ��l��)
    Out.Eye = CameraPosition - Pos;

    // �����ʒu�ւ̍��W�ϊ�
    Pos = mul( Pos, InvMirrorWorldMatrix );
    Pos = TransMirrorPos( Pos ); // �����ϊ�
    Out.VPos = Pos;
    Pos = mul( Pos, MirrorWorldMatrix );

    // �J�������_�̃r���[�ˉe�ϊ�
    Out.Pos = mul( Pos, ViewProjMatrix );
    Out.Pos.x *= -1.0f; // �|���S�������Ԃ�Ȃ��悤�ɍ��E���]�ɂ��ĕ`��

    // �J�����̋��ʂɑ΂��鑊�΍��W
    Out.VCamera = mul( float4(CameraPosition, 1.0f), InvMirrorWorldMatrix );

    // ���_�@��(����������������Ă��邱�Ƃ��l��)
    Out.Normal = normalize( mul( Normal, (float3x3)WorldMatrix ) );

    // �f�B�t���[�Y�F�{�A���r�G���g�F �v�Z
    Out.Color.rgb = AmbientColor;
    if ( !useToon ) {
        Out.Color.rgb += max(0,dot( Out.Normal, -LightDirection )) * DiffuseColor.rgb;
    }
    Out.Color.a = DiffuseColor.a;
    Out.Color = saturate( Out.Color );

    // �e�N�X�`�����W
    Out.Tex = abs(frac(Tex)*0.5);

    if ( useSphereMap ) {
        // �X�t�B�A�}�b�v�e�N�X�`�����W
        float2 NormalWV = mul( Out.Normal, (float3x3)ViewMatrix );
        Out.SpTex.x = NormalWV.x * 0.5f + 0.5f;
        Out.SpTex.y = NormalWV.y * -0.5f + 0.5f;
    }

    return Out;
}

// �s�N�Z���V�F�[�_
float4 BufferShadow_PS(BufferShadow_OUTPUT IN, uniform bool useTexture, uniform bool useSphereMap, uniform bool useToon) : COLOR
{
    // �X�y�L�����F�v�Z
    float3 HalfVector = normalize( normalize(IN.Eye) + -LightDirection );
    float3 Specular = pow( max(0,dot( HalfVector, normalize(IN.Normal) )), SpecularPower ) * SpecularColor;

    float4 Color = IN.Color;
    float4 ShadowColor = float4(AmbientColor, Color.a);  // �e�̐F
    if ( useToon ) {
        ShadowColor = Color;
    }
    if ( useTexture ) {
        // �e�N�X�`���K�p
        float4 TexColor = tex2D( ObjTexSampler, IN.Tex );
        Color *= TexColor;
        ShadowColor *= TexColor;
    }
    if ( useSphereMap ) {
        // �X�t�B�A�}�b�v�K�p
        float4 TexColor = tex2D(ObjSphareSampler,IN.SpTex);
        if(spadd) {
            Color.rgb += TexColor.rgb;
            ShadowColor.rgb += TexColor.rgb;
        } else {
            Color.rgb *= TexColor.rgb;
            ShadowColor.rgb *= TexColor.rgb;
        }
    }
    // �X�y�L�����K�p
    Color.rgb += Specular;

    // �e�N�X�`�����W�ɕϊ�
    IN.ZCalcTex /= IN.ZCalcTex.w;
    float2 TransTexCoord;
    TransTexCoord.x = (1.0f + IN.ZCalcTex.x)*0.5f;
    TransTexCoord.y = (1.0f - IN.ZCalcTex.y)*0.5f;

    if( any( saturate(TransTexCoord) - TransTexCoord ) ) {
        // �V���h�E�o�b�t�@�O
        // ���ʂ̗����ɂ��镔�ʂ͋����\�����Ȃ�
        if(dot((float3)IN.VPos-MirrorPos, MirrorNormal)*dot((float3)IN.VCamera-MirrorPos, MirrorNormal) >= 0.0f) Color.a = 0.0f;
        return Color;
    } else {
        float comp;
        if(parthf) {
            // �Z���t�V���h�E mode2
            comp=1-saturate(max(IN.ZCalcTex.z-tex2D(DefSampler,TransTexCoord).r , 0.0f)*SKII2*TransTexCoord.y-0.3f);
        } else {
            // �Z���t�V���h�E mode1
            comp=1-saturate(max(IN.ZCalcTex.z-tex2D(DefSampler,TransTexCoord).r , 0.0f)*SKII1-0.3f);
        }
        if ( useToon ) {
            // �g�D�[���K�p
            comp = min(saturate(dot(IN.Normal,-LightDirection)*Toon),comp);
            ShadowColor.rgb *= MaterialToon;
        }

        float4 ans = lerp(ShadowColor, Color, comp);
        if( transp ) ans.a = 0.5f;
        // ���ʂ̗����ɂ��镔�ʂ͋����\�����Ȃ�
        if(dot((float3)IN.VPos-MirrorPos, MirrorNormal)*dot((float3)IN.VCamera-MirrorPos, MirrorNormal) >= 0.0f) ans.a = 0.0f;
        return ans;
    }
}

// �I�u�W�F�N�g�`��p�e�N�j�b�N�i�A�N�Z�T���p�j
technique MainTecBS0  < string MMDPass = "object_ss"; bool UseTexture = false; bool UseSphereMap = false; bool UseToon = false; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_3_0 BufferShadowMirror_VS(false, false, false);
        PixelShader  = compile ps_3_0 BufferShadow_PS(false, false, false);
    }
}

technique MainTecBS1  < string MMDPass = "object_ss"; bool UseTexture = true; bool UseSphereMap = false; bool UseToon = false; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_3_0 BufferShadowMirror_VS(true, false, false);
        PixelShader  = compile ps_3_0 BufferShadow_PS(true, false, false);
    }
}

technique MainTecBS2  < string MMDPass = "object_ss"; bool UseTexture = false; bool UseSphereMap = true; bool UseToon = false; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_3_0 BufferShadowMirror_VS(false, true, false);
        PixelShader  = compile ps_3_0 BufferShadow_PS(false, true, false);
    }
}

technique MainTecBS3  < string MMDPass = "object_ss"; bool UseTexture = true; bool UseSphereMap = true; bool UseToon = false; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_3_0 BufferShadowMirror_VS(true, true, false);
        PixelShader  = compile ps_3_0 BufferShadow_PS(true, true, false);
    }
}

// �I�u�W�F�N�g�`��p�e�N�j�b�N�iPMD���f���p�j
technique MainTecBS4  < string MMDPass = "object_ss";
                        bool UseTexture = false; bool UseSphereMap = false; bool UseToon = true; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_3_0 BufferShadowMirror_VS(false, false, true);
        PixelShader  = compile ps_3_0 BufferShadow_PS(false, false, true);
    }
}

technique MainTecBS5  < string MMDPass = "object_ss";
                        bool UseTexture = true; bool UseSphereMap = false; bool UseToon = true; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_3_0 BufferShadowMirror_VS(true, false, true);
        PixelShader  = compile ps_3_0 BufferShadow_PS(true, false, true);
    }
}

technique MainTecBS6  < string MMDPass = "object_ss";
                        bool UseTexture = false; bool UseSphereMap = true; bool UseToon = true; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_3_0 BufferShadowMirror_VS(false, true, true);
        PixelShader  = compile ps_3_0 BufferShadow_PS(false, true, true);
    }
}

technique MainTecBS7  < string MMDPass = "object_ss";
                        bool UseTexture = true; bool UseSphereMap = true; bool UseToon = true; > {
    pass DrawMirrorObject {
        VertexShader = compile vs_3_0 BufferShadowMirror_VS(true, true, true);
        PixelShader  = compile ps_3_0 BufferShadow_PS(true, true, true);
    }
}



///////////////////////////////////////////////////////////////////////////////////////////////
